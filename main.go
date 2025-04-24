package main

import (
	"Gobernetes/manager"
	"Gobernetes/worker"
	"fmt"
)

func main() {
	wHost := "localhost"
	wPorts := []int{10087, 10088, 10089}
	fmt.Println("### Starting g8s workers...")
	for i, wPort := range wPorts {
		w := worker.NewWorker(fmt.Sprintf("worker-%d", i), "memory")
		wApi := worker.WorkerAPI{
			Address: wHost,
			Port:    wPort,
			Worker:  w,
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
	m := manager.NewManager(workers, "", "memory")
	mApi := manager.ManagerAPI{
		Address: mHost,
		Port:    mPort,
		Manager: m,
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	mApi.Start()
}
