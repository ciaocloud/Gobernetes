package worker

import (
	"Gobernetes/stats"
	"Gobernetes/store"
	"Gobernetes/task"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        store.Store
	TaskCount int
	Stats     *stats.Stats
}

func NewWorker(name, taskDbType string) *Worker {
	var db store.Store
	var err error
	switch taskDbType {
	case "in-memory":
		db = store.NewInMemoryTaskStore()
	case "bolt":
		fname := fmt.Sprintf("%s_tasks.db", name)
		db, err = store.NewBoltTaskStore(fname, 0600, "tasks")
		if err != nil {
			log.Fatal(err)
		}
	}
	return &Worker{
		Name:  name,
		Queue: *queue.New(),
		Db:    db,
	}
}

func (w *Worker) AddTask(t *task.Task) {
	w.Queue.Enqueue(*t)
}

func (w *Worker) StartTask(t *task.Task) *task.DockerResult {
	t.StartTime = time.Now()
	config := task.NewConfig(t)
	d := task.NewDocker(config)
	result := d.Run()
	if result.Error != nil {
		log.Printf("[worker] Error starting task %v in %s: %v\n", t.ID, w.Name, result.Error)
		t.State = task.Failed
		w.Db.Put(t.ID.String(), t)
	} else {
		t.ContainerID = result.ContainerId
		t.State = task.Running
		w.Db.Put(t.ID.String(), t)
		log.Printf("[worker] Docker container %s is started for task %v in %s\n", t.ContainerID, t.ID, w.Name)
	}
	return &result
}

func (w *Worker) StopTask(t *task.Task) *task.DockerResult {
	config := task.NewConfig(t)
	d := task.NewDocker(config)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		fmt.Println(w.Name, "[worker] Error stopping container:", result.Error)
	}
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db.Put(t.ID.String(), t)
	log.Printf("[worker] Docker container %s is stopped and removed for task %v in %s\n", t.ContainerID, t.ID, w.Name)
	return &result
}

func (w *Worker) RunTask() *task.DockerResult {
	if w.Queue.Len() == 0 {
		fmt.Printf("[worker] No tasks in the %s's queue\n", w.Name)
		return &task.DockerResult{
			Error: nil,
		}
	}
	taskQueued := w.Queue.Dequeue().(task.Task) // pull a task from worker's queue
	err := w.Db.Put(taskQueued.ID.String(), &taskQueued)
	if err != nil {
		log.Printf("[worker] Error putting task %v into DB: %v\n", taskQueued.ID, err)
		return &task.DockerResult{Error: err}
	}
	val, err := w.Db.Get(taskQueued.ID.String())
	if err != nil {
		log.Printf("[worker] Error getting task %s from DB: %v\n", taskQueued.ID, err)
		return &task.DockerResult{Error: err}
	}
	taskPersisted := *val.(*task.Task)
	var result *task.DockerResult
	if task.ValidateStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(&taskQueued)
		case task.Completed:
			result = w.StopTask(&taskQueued)
		default:
			result.Error = errors.New("[worker] invalid task state while dequeue")
		}
	} else {
		result.Error = fmt.Errorf("[worker] invalid state transition from %v to %v", taskPersisted.State, taskQueued.State)
	}
	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() > 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("[worker] Error running task in %s: %v\n", w.Name, result.Error)
			}
		} else {
			log.Printf("[worker] No tasks in the queue in %s.\n", w.Name)
		}
		log.Println("[worker] Worker RunTasks next check in 10 seconds...")
		time.Sleep(10 * time.Second)
		//log.Printf("### DB: %v\n", len(w.Db))
	}
}

func (w *Worker) GetTasks() []*task.Task {
	tasks, err := w.Db.List()
	if err != nil {
		log.Printf("[worker] Error getting tasks from DB: %v\n", err)
		return nil
	}
	return tasks.([]*task.Task)
}

func (w *Worker) CollectStats() {
	for {
		log.Printf("[worker] %s CollectStats next check in 15 seconds...", w.Name)
		w.Stats = stats.GetStats()
		w.Stats.TaskCount = w.TaskCount
		log.Printf("[worker] %s's stats: %v", w.Name, w.Stats)
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) InspectTask(t *task.Task) task.DockerInspectResponse {
	cfg := task.NewConfig(t)
	d := task.NewDocker(cfg)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Printf("[worker] %s checking status of tasks\n", w.Name)
		tasks, err := w.Db.List()
		if err != nil {
			log.Printf("[worker] Error getting tasks from DB: %v\n", err)
			return
		}
		for _, t := range tasks.([]*task.Task) {
			if t.State == task.Running {
				taskId := t.ID.String()
				inspect := w.InspectTask(t)
				if inspect.Error != nil {
					log.Printf("[worker] Error inspecting task %s: %v\n", taskId, inspect.Error)
				}
				if inspect.InspectResponse == nil {
					log.Printf("[worker] No container for running task %s\n", t.ID.String())
					t.State = task.Failed
					w.Db.Put(taskId, t)
				}
				if inspect.InspectResponse.State.Status == "exited" {
					log.Printf("[worker] Container for running task %s is exited in non-running state %s\n", taskId, inspect.InspectResponse.State.Status)
					t.State = task.Failed
					w.Db.Put(taskId, t)
				}
				t.HostPorts = inspect.InspectResponse.NetworkSettings.Ports
				w.Db.Put(taskId, t)
			}
		}
		log.Printf("[worker] %s UpdateTasks complete. Next check in 15 seconds...\n", w.Name)
		time.Sleep(15 * time.Second)
	}
}
