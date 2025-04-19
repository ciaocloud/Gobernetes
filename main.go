package main

import (
	"Gobernetes/manager"
	"Gobernetes/task"
	"Gobernetes/worker"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"log"
	"time"
)

func main() {
	host := "localhost"
	port := 10086

	fmt.Println("Starting worker...")
	w := worker.Worker{
		Name:  "test-worker",
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.WorkerAPI{
		Address: host,
		Port:    port,
		Worker:  &w,
	}
	go func(w *worker.Worker) {
		for {
			if w.Queue.Len() > 0 {
				result := w.RunTask()
				if result.Error != nil {
					log.Printf("Error running task: %v\n", result.Error)
				}
			} else {
				log.Printf("No tasks in the queue. \n")
			}
			log.Printf("Next check in 10 seconds...\n")
			time.Sleep(10 * time.Second)
			log.Printf("### DB: %v\n", len(w.Db))
		}
	}(&w)
	go api.Start()

	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	mgr := manager.NewManager(workers)
	for i := 0; i < 3; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}
		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}
		mgr.AddTask(te)
		mgr.SendWork()
	}

	go func() {
		for {
			fmt.Printf("[Manager] Updating tasks from %d workers\n", len(mgr.Workers))
			mgr.UpdateTasks()
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		for _, t := range mgr.TaskDb {
			fmt.Printf("[Manager] Task ID: %v, State: %d, Worker: %s\n", t.ID, t.State, mgr.TaskWorkerMap[t.ID])
			time.Sleep(15 * time.Second)
		}
	}
}

//func main() {
//	t := task.Task{
//		ID:     uuid.New(),
//		Name:   "task1",
//		State:  task.Pending,
//		Image:  "image1",
//		Memory: 1024,
//		Disk:   1,
//	}
//
//	te := task.TaskEvent{
//		ID:        uuid.New(),
//		State:     task.Pending,
//		Timestamp: time.Now(),
//		Task:      t,
//	}
//
//	fmt.Printf("Task: %v\n", t)
//	fmt.Printf("Task Event: %v\n", te)
//
//	w := worker.Worker{
//		Name:      "worker-1",
//		Queue:     *queue.New(),
//		Db:        make(map[uuid.UUID]*task.Task),
//		TaskCount: 0,
//	}
//	fmt.Printf("Worker: %v\n", w)
//	w.CollectStats()
//	w.RunTask()
//	w.StartTask()
//	w.StopTask()
//
//	m := manager.Manager{
//		Pending: *queue.New(),
//		TaskDb:  make(map[string]*task.Task),
//		EventDb: make(map[string]*task.TaskEvent),
//		Workers: []string{w.Name},
//	}
//	fmt.Printf("Manager: %v\n", m)
//	m.SelectWorker()
//	m.UpdateTasks()
//	m.SendWork()
//
//	n := node.Node{
//		Name:   "node-1",
//		Ip:     "192.168.1.1",
//		Cores:  4,
//		Memory: 1024,
//		Disk:   25,
//		Role:   "worker",
//	}
//	fmt.Printf("Node: %v\n", n)
//
//	dockerTask, createResult := createContainer()
//	if createResult.Error != nil {
//		fmt.Println("Error creating container:", createResult.Error)
//		os.Exit(1)
//	}
//	time.Sleep(5 * time.Second)
//	stopContainer(dockerTask, createResult.ContainerId)
//
//}
//
//func createContainer() (*task.Docker, *task.DockerResult) {
//	c := task.Config{
//		Name:  "test-container",
//		Image: "postgres:13",
//		Env:   []string{"POSTGRES_USER=cube", "POSTGRES_PASSWORD=secret"},
//	}
//	dc, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
//	d := task.Docker{
//		Client: dc,
//		Config: c,
//	}
//	result := d.Run()
//	if result.Error != nil {
//		fmt.Println(result.Error)
//		return nil, nil
//	}
//	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, c)
//	return &d, &result
//}
//
//func stopContainer(d *task.Docker, id string) *task.DockerResult {
//	result := d.Stop(id)
//	if result.Error != nil {
//		fmt.Println(result.Error)
//		return nil
//	}
//	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)
//	return &result
//}
