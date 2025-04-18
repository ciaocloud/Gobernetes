package task

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateAndStopContainer(t *testing.T) {
	c := Config{
		Name:  "test-container",
		Image: "postgres:13",
		Env:   []string{"POSTGRES_USER=cube", "POSTGRES_PASSWORD=secret"},
	}
	d, err := NewDocker(&c)
	require.NoError(t, err)
	result := d.Run()
	require.NoError(t, result.Error)
	containerId := result.ContainerId
	fmt.Printf("Container %s is running with config %v\n", containerId, c)

	result = d.Stop(containerId)
	require.NoError(t, result.Error)
	fmt.Printf("Container %s has been stopped and removed\n", containerId)
}
