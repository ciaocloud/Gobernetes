package manager

import (
	"Gobernetes/task"
	"Gobernetes/worker"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	LastWorker int
}

func NewManager(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	for i := range workers {
		workerTaskMap[workers[i]] = []uuid.UUID{}
	}
	taskWorkerMap := make(map[uuid.UUID]string)
	return &Manager{
		Pending:       *queue.New(),
		TaskDb:        taskDb,
		EventDb:       eventDb,
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) SelectWorker() string {
	// Select the next worker in a round-robin fashion
	// Worker string in the format of <hostname>:<port>
	var nextWorker int
	if m.LastWorker+1 < len(m.Workers) {
		nextWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		nextWorker = 0
		m.LastWorker = 0
	}
	return m.Workers[nextWorker]
}

func (m *Manager) SendWork() {
	// check if there are any tasks in the queue
	if m.Pending.Len() == 0 {
		log.Println("No work in the queue")
		return
	}
	// pull a task from manager's queue
	task_event := m.Pending.Dequeue().(task.TaskEvent)
	t := task_event.Task
	w := m.SelectWorker()

	// perform administrative work for the manager to keep track of which worker the task is assigned to
	m.EventDb[task_event.ID] = &task_event
	m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], t.ID)
	m.TaskWorkerMap[t.ID] = w

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
	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error posting task event to %v: %v\n", w, err)
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
