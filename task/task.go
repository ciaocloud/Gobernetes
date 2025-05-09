package task

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID
	ContainerID string
	Name        string
	State       State

	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	ExposedPort   nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string

	StartTime  time.Time
	FinishTime time.Time

	HealthCheck  string
	RestartCount int
	HostPorts    nat.PortMap
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

// Config is a struct that holds configuration settings for the Docker container.
type Config struct {
	// identifiers
	Name  string // Name of the container/task
	Image string // Docker image to use to run the container

	// resource allocation
	Cpu           float64
	Memory        int64    // Memory limit in MB
	Disk          int64    // Disk limit in GB
	Env           []string // Environment variables to set in the container
	Cmd           []string // Command to run in the container (optional)
	RestartPolicy string   // Restart policy for the container: ["", "always", "on-failure", "unless-stopped"]

	// network settings
	ExposedPort  nat.PortSet // list of ports to expose from the container
	AttachStdin  bool        // if stdin should be attached
	AttachStdout bool        // if stdout should be attached
	AttachStderr bool        // if stderr should be attached
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		Image:         t.Image,
		Cpu:           t.Cpu,
		Memory:        t.Memory,
		Disk:          t.Disk,
		ExposedPort:   t.ExposedPort,
		RestartPolicy: t.RestartPolicy,
	}
}

type Docker struct {
	Client *client.Client
	Config Config
}

func NewDocker(c *Config) *Docker {
	d_client, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return &Docker{
		Client: d_client,
		Config: *c,
	}
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	// Pull the image
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})
	if err != nil {
		log.Printf("Failed to pull image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		log.Printf("Failed to read image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	// Create the container
	cc := container.Config{
		Image:        d.Config.Image,
		ExposedPorts: d.Config.ExposedPort,
		Env:          d.Config.Env,
		Tty:          false,
	}
	hc := container.HostConfig{
		Resources: container.Resources{
			Memory: d.Config.Memory * 1024 * 1024,
		},
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(d.Config.RestartPolicy),
		},
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf("Failed to create container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	// Start the container
	if err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Printf("Failed to start container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	// Get the container logs
	out, err := d.Client.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		log.Printf("Failed to get logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	if _, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
		log.Printf("Failed to copy logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}
	return DockerResult{
		ContainerId: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	ctx := context.Background()
	log.Printf("Stopping container %v\n", id)
	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		log.Printf("Failed to stop container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}
	if err = d.Client.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	}); err != nil {
		log.Printf("Failed to remove container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}
	return DockerResult{
		Action: "stop",
		Result: "success",
	}
}

type DockerInspectResponse struct {
	*container.InspectResponse
	Error error
}

func (d *Docker) Inspect(containerID string) DockerInspectResponse {
	ctx := context.Background()
	inspect, err := d.Client.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("Failed to inspect container %s: %v\n", containerID, err)
		return DockerInspectResponse{Error: err}
	}
	return DockerInspectResponse{
		InspectResponse: &inspect,
	}
}
