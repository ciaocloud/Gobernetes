package manager

import (
	"Gobernetes/node"
	"Gobernetes/scheduler"
	"Gobernetes/task"
	"Gobernetes/worker"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

/**
 * Manager is responsible for managing the workers and tasks.
 * It is responsible for the following:
 * 1. Handling requests from users
 * 2. Assigning tasks to workers, i.e., scheduling
 * 3. Keeping track of task states and worker states
 * 4. Restarting failed tasks
 * 5. Handling errors
 *
 * In Borg: BorgMaster
 * In Kubernetes, it is the control plane, including: KubeScheduler, KubeControllerManager, API Server, etcd.
 * In HashiCorp Nomad: Nomad Server
 */

type Manager struct {
	Pending queue.Queue
	TaskDb  map[uuid.UUID]*task.Task
	EventDb map[uuid.UUID]*task.TaskEvent

	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string

	WorkerNodes []*node.Node
	Scheduler   scheduler.Scheduler
}

func NewManager(workers []string, schedulerType string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)

	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	for i := range workers {
		workerTaskMap[workers[i]] = []uuid.UUID{}
	}

	var s scheduler.Scheduler
	switch schedulerType {
	case "RoundRobin":
		s = &scheduler.RoundRobin{Name: "RoundRobin"}
	case "EPVM":
		s = &scheduler.EPVM{Name: "EPVM"}
	default:
		s = &scheduler.RoundRobin{Name: "EPVM"}
	}

	return &Manager{
		Pending:       *queue.New(),
		TaskDb:        taskDb,
		EventDb:       eventDb,
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		Scheduler:     s,
	}
}

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)
	if candidates == nil {
		msg := fmt.Sprintf("No available worker node candidates match resource request for task %v", t.ID)
		return nil, errors.New(msg)
	}
	scores := m.Scheduler.Score(t, candidates)
	if scores == nil {
		msg := fmt.Sprintf("No available worker node candidates match resource request for task %v", t.ID)
		return nil, errors.New(msg)
	}
	selectedNode := m.Scheduler.Pick(scores, candidates)
	return selectedNode, nil
}

func (m *Manager) SendWork() {
	// check if there are any tasks in the queue
	if m.Pending.Len() == 0 {
		log.Println("No work in the queue")
		return
	}
	// pull a task from manager's queue
	task_event := m.Pending.Dequeue().(task.TaskEvent)
	m.EventDb[task_event.ID] = &task_event
	log.Printf("[manager] Pulled task event %v from pending queue\n", task_event.ID)

	t := task_event.Task
	// check if the task is an existing one, i.e., already in the task worker map
	if tw, ok := m.TaskWorkerMap[t.ID]; ok {
		persistedTask := m.TaskDb[t.ID]
		if task_event.State == task.Completed && task.ValidateStateTransition(persistedTask.State, task_event.State) {
			// if the state of the task from the pending queue is completed,
			// and the running task can be transitioned to completed from its current state,
			// we stop the running task.
			m.stopTask(tw, t.ID.String())
			return
		}
	}

	w, err := m.SelectWorker(t)
	if err != nil {
		log.Printf("Error selecting worker for task %v: %v\n", t.ID, err)
		return
	}

	log.Printf("[manager] Selected worker %s for task %s\n", w.Name, t.ID)
	// perform administrative work for the manager to keep track of which worker the task is assigned to
	m.WorkerTaskMap[w.Name] = append(m.WorkerTaskMap[w.Name], t.ID)
	m.TaskWorkerMap[t.ID] = w.Name

	// set task state to scheduled
	t.State = task.Scheduled
	m.TaskDb[t.ID] = &t

	// encode task event to JSON
	data, err := json.Marshal(task_event)
	if err != nil {
		log.Printf("Error marshalling task event %v: %v\n", t, err)
		return
	}
	// send task event to the selected worker
	url := fmt.Sprintf("http://%s/tasks", w.Name)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error posting task event to %v: %v\n", w.Name, err)
		m.Pending.Enqueue(task_event)
		return
	}
	defer resp.Body.Close()
	// check the response from the worker
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		errResp := worker.ErrResponse{}
		if err := decoder.Decode(&errResp); err != nil {
			fmt.Println("Error decoding error response:", err)
			return
		}
		log.Printf("Error response (%D) from worker %s: %s\n", errResp.HttpStatus, w, errResp.Message)
		return
	}

	t = task.Task{}
	if err := decoder.Decode(&t); err != nil {
		log.Printf("Error decoding task response: %v\n", err)
		return
	}
	log.Printf("Task %v has been scheduled: %v\n", t.ID, t.State)
}

func (m *Manager) UpdateTasksOnce() {
	// Update the task state in the task database
	for _, w := range m.Workers {
		// for each worker:
		// 1. query the worker to get a list of its tasks
		// 2. for each task, update its state in the manager's database so it matches the state from the worker
		log.Printf("Checking worker %s for task updates\n", w)
		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Get(url) // query the worker to get a list of its tasks
		if err != nil {
			log.Printf("Error getting tasks from worker %s: %v\n", w, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("Error response from worker %s: %v\n", w, resp.Status)
		}
		decoder := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		if err := decoder.Decode(&tasks); err != nil {
			log.Printf("Error unmarshalling tasks from worker %s: %v\n", w, err)
		}
		for _, t := range tasks {
			log.Printf("Attempt updating task %s\n", t.ID)
			task_in_db, ok := m.TaskDb[t.ID]
			if !ok {
				log.Printf("Task %s does not exist\n", t.ID)
				return
			}
			if task_in_db.State != t.State {
				task_in_db.State = t.State
			}
			task_in_db.StartTime = t.StartTime
			task_in_db.FinishTime = t.FinishTime
			task_in_db.ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) UpdateTasks() {
	for {
		log.Printf("Checking workers for task updates\n")
		m.UpdateTasksOnce()
		log.Println("Task update completed, manager UpdateTasks thread sleeping for 15 seconds...")
		time.Sleep(15 * time.Second)
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the manager's queue")
		m.SendWork()
		log.Println("Tasks sending to workers completed, manager ProcessTasks thread sleeping for 10 seconds...")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
}

func (m *Manager) DoHealthChecks() {
	for {
		log.Println("Checking health checks...")
		for _, t := range m.GetTasks() {
			if t.State == task.Running && t.RestartCount < 3 {
				err := m.checkTaskHealth(t)
				if err != nil && t.RestartCount < 3 {
					m.restartTask(t)
				}
			} else if t.State == task.Failed && t.RestartCount < 3 {
				m.restartTask(t)
			}
		}
		log.Println("Manager health check completed, manager DoHealthChecks thread sleeping for 60 seconds...")
		time.Sleep(60 * time.Second)
	}
}

func (m *Manager) checkTaskHealth(t *task.Task) error {
	log.Printf("Checking health of task %s: %s\n", t.ID, t.HealthCheck)
	w := m.TaskWorkerMap[t.ID]
	wIp := strings.Split(w, ":")[0]
	hostPort := getHostPort(t.HostPorts)
	url := fmt.Sprintf("http://%s:%s%s", wIp, *hostPort, t.HealthCheck)
	resp, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Error connecting health check for task %s: %v\n", t.ID, err)
		log.Println(msg)
		return errors.New(msg)
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Health check failed for task %s: %s\n", t.ID, resp.Status)
		log.Println(msg)
		return errors.New(msg)
	}
	log.Printf("Health check successful for task %s: %s\n", t.ID, resp.Status)
	return nil
}

func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
		return &ports[k][0].HostPort
	}
	return nil
}

func (m *Manager) restartTask(t *task.Task) {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb[t.ID] = t

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}
	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Error marshalling task event: %v\n", err)
		return
	}
	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(t)
		return
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		errResp := worker.ErrResponse{}
		if err := decoder.Decode(&errResp); err != nil {
			fmt.Println("Error decoding error response:", err)
			return
		}
		log.Printf("Error response (%D) from worker %s: %s\n", errResp.HttpStatus, w, errResp.Message)
		return
	}
	newTask := task.Task{}
	if err := decoder.Decode(&newTask); err != nil {
		log.Printf("Error decoding task response: %v\n", err)
		return
	}
}

func (m *Manager) stopTask(worker string, taskID string) {
	url := fmt.Sprintf("http://%s/tasks/%s", worker, taskID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("Error creating request to stop task %s on worker %s: %v\n", taskID, worker, err)
		return
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error connecting %s to stopp task %s on worker %s: %v\n", url, taskID, worker, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from worker %s: %v\n", worker, resp.Status)
		return
	}
	log.Printf("Successfully stopped task %s on worker %s\n", taskID, worker)
}
