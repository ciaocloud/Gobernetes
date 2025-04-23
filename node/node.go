package node

import (
	"Gobernetes/stats"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Node struct {
	Name            string
	Ip              string
	Cores           int
	Memory          int64
	MemoryAllocated int64
	Disk            int64
	DiskAllocated   int64
	Role            string
	TaskCount       int
	Api             string
	Stats           stats.Stats
}

func NewNode(name string, api string, role string) *Node {
	return &Node{
		Name: name,
		Api:  api,
		Role: role,
	}
}

func (n *Node) GetStats() (*stats.Stats, error) {
	var resp *http.Response
	var err error
	url := fmt.Sprintf("%s/stats", n.Api)
	resp, err = HttpWithRetry(http.Get, url, 10)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to node %s %v: %v\n", n.Name, n.Api, err)
		log.Println(msg)
		return nil, errors.New(msg)
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Error getting stats from node %s %v: %s\n", n.Name, n.Api, resp.Status)
		log.Println(msg)
		return nil, errors.New(msg)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var stats stats.Stats
	if err := json.Unmarshal(body, &stats); err != nil {
		msg := fmt.Sprintf("Error unmarshalling stats from node %s %v: %v\n", n.Name, n.Api, err)
		log.Println(msg)
		return nil, errors.New(msg)
	}
	if stats.MemStat == nil || stats.DiskStat == nil {
		return nil, fmt.Errorf("Error getting stats from node", n.Name)
	}
	n.Memory = int64(stats.MemStat.Total)
	n.Disk = int64(stats.DiskStat.Total)
	n.Stats = stats
	return &n.Stats, nil
}

func HttpWithRetry(f func(string) (*http.Response, error), url string, count int) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < count; i++ {
		resp, err = f(url)
		if err == nil {
			return resp, nil
		} else {
			fmt.Printf("Error calling url %v, retrying...\n", url)
			time.Sleep(5 * time.Second)
		}
	}
	return nil, err
}
