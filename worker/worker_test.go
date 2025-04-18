package worker

import (
	"Gobernetes/task"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	db := make(map[uuid.UUID]*task.Task)
	testWorker := Worker{
		Name:  "worker1",
		Db:    db,
		Queue: *queue.New(),
	}
	testTask := &task.Task{
		ID:    uuid.New(),
		Name:  "test-task",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}
	fmt.Println("starting task")
	testWorker.AddTask(testTask)
	result := testWorker.RunTask()
	if result.Error != nil {
		t.Errorf("Error running task: %v", result.Error)
	}

	testTask.ContainerID = result.ContainerId
	fmt.Printf("Task %s is running with container ID %s\n", testTask.ID, testTask.ContainerID)
	time.Sleep(10 * time.Second) // Simulate some work

	fmt.Println("stopping task")
	testTask.State = task.Completed
	testWorker.AddTask(testTask)
	result = testWorker.RunTask()
	if result.Error != nil {
		t.Errorf("Error stopping task: %v", result.Error)
	}
}
