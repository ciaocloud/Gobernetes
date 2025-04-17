package main

import (
	"Gobernetes/manager"
	"Gobernetes/node"
	"Gobernetes/task"
	"Gobernetes/worker"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"time"
)

func main() {
	t := task.Task{
		ID:     uuid.New(),
		Name:   "task1",
		State:  task.Pending,
		Image:  "image1",
		Memory: 1024,
		Disk:   1,
	}

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Pending,
		Timestamp: time.Now(),
		Task:      t,
	}

	fmt.Printf("Task: %v\n", t)
	fmt.Printf("Task Event: %v\n", te)

	w := worker.Worker{
		Name:      "worker-1",
		Queue:     *queue.New(),
		Db:        make(map[uuid.UUID]*task.Task),
		TaskCount: 0,
	}
	fmt.Printf("Worker: %v\n", w)
	w.CollectStats()
	w.RunTask()
	w.StartTask()
	w.StopTask()

	m := manager.Manager{
		Pending: *queue.New(),
		TaskDb:  make(map[string]*task.Task),
		EventDb: make(map[string]*task.TaskEvent),
		Workers: []string{w.Name},
	}
	fmt.Printf("Manager: %v\n", m)
	m.SelectWorker()
	m.UpdateTasks()
	m.SendWork()

	n := node.Node{
		Name:   "node-1",
		Ip:     "192.168.1.1",
		Cores:  4,
		Memory: 1024,
		Disk:   25,
		Role:   "worker",
	}
	fmt.Printf("Node: %v\n", n)

}
