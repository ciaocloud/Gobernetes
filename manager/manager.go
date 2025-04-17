package manager

import (
	"Gobernetes/task"
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending queue.Queue
	TaskDb  map[string]*task.Task
	EventDb map[string]*task.TaskEvent

	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

//func NewManager() *Manager {
//	return &Manager{}
//}

func (m *Manager) SelectWorker() {
	fmt.Println("Selecting workers")
}

func (m *Manager) UpdateTasks() {
	fmt.Println("Updating tasks")
}

func (m *Manager) SendWork() {
	fmt.Println("Sending work")
}
