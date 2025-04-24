package store

import (
	"Gobernetes/task"
	"fmt"
)

type Store interface {
	Get(key string) (interface{}, error)
	Put(key string, value interface{}) error
	List() (interface{}, error)
	Count() (int, error)
}

type InMemoryTaskStore struct {
	Db map[string]*task.Task
}

func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		Db: make(map[string]*task.Task),
	}
}

func (imts *InMemoryTaskStore) Get(key string) (interface{}, error) {
	t, ok := imts.Db[key]
	if !ok {
		return nil, fmt.Errorf("task %s not found", key)
	}
	return t, nil
}

func (imts *InMemoryTaskStore) Put(key string, value interface{}) error {
	t, ok := value.(*task.Task)
	if !ok {
		return fmt.Errorf("value %v is not a task type", value)
	}
	imts.Db[key] = t
	return nil
}

func (imts *InMemoryTaskStore) List() (interface{}, error) {
	var tasks []*task.Task
	for _, t := range imts.Db {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (imts *InMemoryTaskStore) Count() (int, error) {
	return len(imts.Db), nil
}

type InMemoryTaskEventStore struct {
	Db map[string]*task.TaskEvent
}

func NewInMemoryTaskEventStore() *InMemoryTaskEventStore {
	return &InMemoryTaskEventStore{
		Db: make(map[string]*task.TaskEvent),
	}
}

func (imtes *InMemoryTaskEventStore) Get(key string) (interface{}, error) {
	t, ok := imtes.Db[key]
	if !ok {
		return nil, fmt.Errorf("task event %s not found", key)
	}
	return t, nil
}

func (imtes *InMemoryTaskEventStore) Put(key string, value interface{}) error {
	t, ok := value.(*task.TaskEvent)
	if !ok {
		return fmt.Errorf("value %v is not a task event type", value)
	}
	imtes.Db[key] = t
	return nil
}

func (imtes *InMemoryTaskEventStore) List() (interface{}, error) {
	var taskEvents []*task.TaskEvent
	for _, t := range imtes.Db {
		taskEvents = append(taskEvents, t)
	}
	return taskEvents, nil
}

func (imtes *InMemoryTaskEventStore) Count() (int, error) {
	return len(imtes.Db), nil
}
