package main

import (
	"Gobernetes/manager"
	"Gobernetes/task"
	"Gobernetes/worker"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	wHost := "localhost"
	wPort := 10086
	fmt.Println("Starting g8s worker...")
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	wApi := worker.WorkerAPI{
		Address: wHost,
		Port:    wPort,
		Worker:  &w,
	}
	go w.RunTasks()
	go wApi.Start()

	mHost := "localhost"
	mPort := 10087
	workers := []string{fmt.Sprintf("%s:%d", wHost, wPort)}
	fmt.Println("Starting g8s manager...")
	m := manager.NewManager(workers)
	mApi := manager.ManagerAPI{
		Address: mHost,
		Port:    mPort,
		Manager: m,
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	mApi.Start()
}
