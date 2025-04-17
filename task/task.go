package task

import (
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"time"
)

type Status int

const (
	Pending Status = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID    uuid.UUID
	Name  string
	State Status

	Image         string
	Memory        int
	Disk          int
	ExposedPort   nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string

	StartTime time.Time
	EndTime   time.Time
}

type TaskEvent struct {
	ID        uuid.UUID
	State     Status
	Timestamp time.Time
	Task      Task
}
