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
}

func (w *Worker) AddTask(t *task.Task) {
	w.Queue.Enqueue(*t)
}

func (w *Worker) StartTask(t *task.Task) *task.DockerResult {
	t.StartTime = time.Now()
	config := task.NewConfig(t)
	d, _ := task.NewDocker(config)
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
	d, err := task.NewDocker(config)
	if err != nil {
		log.Printf("Error creating Docker client: %v\n", err)
	}
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

func (w *Worker) CollectStats() {
	fmt.Println("Collecting stats")
}
