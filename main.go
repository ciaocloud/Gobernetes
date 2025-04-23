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
	wPorts := []int{10087, 10088, 10089}
	fmt.Println("### Starting g8s workers...")
	for i, wPort := range wPorts {
		w := worker.Worker{
			Name:  fmt.Sprintf("worker-%d", i),
			Queue: *queue.New(),
			Db:    make(map[uuid.UUID]*task.Task),
		}
		wApi := worker.WorkerAPI{
			Address: wHost,
			Port:    wPort,
			Worker:  &w,
		}
		go w.RunTasks()
		go w.CollectStats()
		go wApi.Start()
	}

	mHost := "localhost"
	mPort := 10086
	workers := make([]string, len(wPorts))
	for i, wPort := range wPorts {
		workers[i] = fmt.Sprintf("%s:%d", wHost, wPort)
	}

	fmt.Println("### Starting g8s manager...")
	m := manager.NewManager(workers, "RoundRobin")
	mApi := manager.ManagerAPI{
		Address: mHost,
		Port:    mPort,
		Manager: m,
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	mApi.Start()
}
