package worker

import (
	"Gobernetes/task"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
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
		log.Printf("Error starting task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = t
	} else {
		t.ContainerID = result.ContainerId
		t.State = task.Running
		w.Db[t.ID] = t
		log.Printf("Docker container %s is started for task %v\n", t.ContainerID, t.ID)
	}
	return &result
}

func (w *Worker) StopTask(t *task.Task) *task.DockerResult {
	config := task.NewConfig(t)
	d := task.NewDocker(config)
	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		fmt.Println("Error stopping container:", result.Error)
	}
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = t
	log.Printf("Docker container %s is stopped and removed for task %v\n", t.ContainerID, t.ID)
	return &result
}

func (w *Worker) RunTask() *task.DockerResult {
	if w.Queue.Len() == 0 {
		fmt.Println("No tasks in the queue")
		return &task.DockerResult{
			Error: nil,
		}
	}
	taskQueued := w.Queue.Dequeue().(task.Task) // pull a task from worker's queue
	taskPersisted := w.Db[taskQueued.ID]        // retrieve the task from worker's database
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = taskPersisted
	}

	var result *task.DockerResult
	if task.ValidateStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(&taskQueued)
		case task.Completed:
			result = w.StopTask(&taskQueued)
		default:
			result.Error = errors.New("invalid task state while dequeue")
		}
	} else {
		result.Error = fmt.Errorf("invalid state transition from %v to %v", taskPersisted.State, taskQueued.State)
	}
	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() > 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks in the queue.\n")
		}
		log.Println("Worker RunTasks next check in 10 seconds...")
		time.Sleep(10 * time.Second)
		//log.Printf("### DB: %v\n", len(w.Db))
	}
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := make([]*task.Task, 0, len(w.Db))
	for _, t := range w.Db {
		tasks = append(tasks, t)
	}
	return tasks
}

func (w *Worker) CollectStats() {
	for {
		log.Println("CollectStats next check in 15 seconds...")
		w.Stats = GetStats()
		w.Stats.TaskCount = w.TaskCount
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
		log.Println("Checking status of tasks")
		for id, t := range w.Db {
			if t.State == task.Running {
				inspect := w.InspectTask(t)
				if inspect.Error != nil {
					log.Printf("Error inspecting task %s: %v\n", id, inspect.Error)
				}
				if inspect.InspectResponse == nil {
					log.Printf("No container for running task %s\n", id)
					w.Db[id].State = task.Failed
				}
				if inspect.InspectResponse.State.Status == "exited" {
					log.Printf("Container for running task %s is exited in non-running state %s\n", id, inspect.InspectResponse.State.Status)
					w.Db[id].State = task.Failed
				}
				w.Db[id].HostPorts = inspect.InspectResponse.NetworkSettings.Ports
			}
		}
		log.Println("UpdateTasks complete. Next check in 15 seconds...")
		time.Sleep(15 * time.Second)
	}
}
