package store

import (
	"Gobernetes/task"
	"encoding/json"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
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

type InMemoryEventStore struct {
	Db map[string]*task.TaskEvent
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		Db: make(map[string]*task.TaskEvent),
	}
}

func (imes *InMemoryEventStore) Get(key string) (interface{}, error) {
	t, ok := imes.Db[key]
	if !ok {
		return nil, fmt.Errorf("task event %s not found", key)
	}
	return t, nil
}

func (imes *InMemoryEventStore) Put(key string, value interface{}) error {
	t, ok := value.(*task.TaskEvent)
	if !ok {
		return fmt.Errorf("value %v is not a task event type", value)
	}
	imes.Db[key] = t
	return nil
}

func (imes *InMemoryEventStore) List() (interface{}, error) {
	var taskEvents []*task.TaskEvent
	for _, t := range imes.Db {
		taskEvents = append(taskEvents, t)
	}
	return taskEvents, nil
}

func (imes *InMemoryEventStore) Count() (int, error) {
	return len(imes.Db), nil
}

type BoltTaskStore struct {
	Db       *bolt.DB
	DbFile   string
	FileMode os.FileMode
	Bucket   string
}

func NewBoltTaskStore(dbFile string, fileMode os.FileMode, bucket string) (*BoltTaskStore, error) {
	db, err := bolt.Open(dbFile, fileMode, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %v", err)
	}
	defer db.Close()
	bts := &BoltTaskStore{
		Db:       db,
		DbFile:   dbFile,
		FileMode: fileMode,
		Bucket:   bucket,
	}
	err = bts.Db.Update(func(tx *bolt.Tx) error {
		//_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		_, err := tx.CreateBucket([]byte(bucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %v", bucket, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bts, nil
}

func (bts *BoltTaskStore) Get(key string) (interface{}, error) {
	var taskValue task.Task
	err := bts.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bts.Bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bts.Bucket)
		}
		v := b.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("task %s not found", key)
		}
		err := json.Unmarshal(v, &taskValue)
		if err != nil {
			return fmt.Errorf("failed to unmarshal task %s: %v", key, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &taskValue, nil
}

func (bts *BoltTaskStore) Put(key string, value interface{}) error {
	taskValue, ok := value.(*task.Task)
	if !ok {
		return fmt.Errorf("value %v is not a task type", value)
	}
	err := bts.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bts.Bucket))
		buf, err := json.Marshal(taskValue)
		if err != nil {
			return fmt.Errorf("failed to marshal task %s: %v", key, err)
		}
		err = b.Put([]byte(key), buf)
		if err != nil {
			return fmt.Errorf("failed to put task %s: %v", key, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (bts *BoltTaskStore) Close() {
	bts.Db.Close()
}

func (bts *BoltTaskStore) Count() (int, error) {
	count := 0
	err := bts.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
		//	Or simply: count = b.Stats().KeyN
		return nil
	})
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (bts *BoltTaskStore) List() (interface{}, error) {
	var tasks []*task.Task
	err := bts.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bts.Bucket))
		b.ForEach(func(k, v []byte) error {
			var taskValue task.Task
			err := json.Unmarshal(v, &taskValue)
			if err != nil {
				return err
			}
			tasks = append(tasks, &taskValue)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

type BoltEventStore struct {
	Db       *bolt.DB
	DbFile   string
	FileMode os.FileMode
	Bucket   string
}

func NewBoltEventStore(dbFile string, fileMode os.FileMode, bucket string) (*BoltEventStore, error) {
	db, err := bolt.Open(dbFile, fileMode, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %v", err)
	}
	defer db.Close()
	bes := &BoltEventStore{
		Db:       db,
		DbFile:   dbFile,
		FileMode: fileMode,
		Bucket:   bucket,
	}
	err = bes.Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(bucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %v", bucket, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bes, nil
}

func (bes *BoltEventStore) Close() {
	bes.Db.Close()
}

func (bes *BoltEventStore) Get(key string) (interface{}, error) {
	var taskEventValue task.TaskEvent
	err := bes.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bes.Bucket))
		v := b.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("task event %s not found", key)
		}
		err := json.Unmarshal(v, &taskEventValue)
		if err != nil {
			return fmt.Errorf("failed to unmarshal task event %s: %v", key, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &taskEventValue, nil
}

func (bes *BoltEventStore) Put(key string, value interface{}) error {
	taskEventValue, ok := value.(*task.TaskEvent)
	if !ok {
		return fmt.Errorf("value %v is not a task event type", value)
	}
	err := bes.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bes.Bucket))
		buf, err := json.Marshal(taskEventValue)
		if err != nil {
			return fmt.Errorf("failed to marshal task event %s: %v", key, err)
		}
		err = b.Put([]byte(key), buf)
		if err != nil {
			return fmt.Errorf("failed to put task event %s: %v", key, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (bes *BoltEventStore) Count() (int, error) {
	count := 0
	err := bes.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bes.Bucket))
		b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
		return nil
	})
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (bes *BoltEventStore) List() (interface{}, error) {
	var taskEvents []*task.TaskEvent
	err := bes.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bes.Bucket))
		b.ForEach(func(k, v []byte) error {
			var taskEventValue task.TaskEvent
			err := json.Unmarshal(v, &taskEventValue)
			if err != nil {
				return err
			}
			taskEvents = append(taskEvents, &taskEventValue)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return taskEvents, nil
}
