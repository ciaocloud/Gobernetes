package scheduler

import (
	"Gobernetes/node"
	"Gobernetes/task"
	"math"
	"time"
)

const (
	// LIEB square ice constant
	// https://en.wikipedia.org/wiki/Lieb%27s_square_ice_constant
	LIEB = 1.53960071783900203869
)

type Scheduler interface {
	SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node
	Score(t task.Task, nodes []*node.Node) map[string]float64
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}

type RoundRobin struct {
	Name       string
	LastWorker int
}

func (rr *RoundRobin) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return nodes
}

func (rr *RoundRobin) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	var nextWorker int
	if rr.LastWorker+1 < len(nodes) {
		nextWorker = rr.LastWorker + 1
		rr.LastWorker++
	} else {
		nextWorker = 0
		rr.LastWorker = 0
	}

	scores := make(map[string]float64)
	for i, node := range nodes {
		if i == nextWorker {
			scores[node.Name] = 0.1
		} else {
			scores[node.Name] = 1.0
		}
	}
	return scores
}

func (rr *RoundRobin) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var bestScore float64
	for i, node := range candidates {
		if i == 0 {
			bestNode = node
			bestScore = scores[node.Name]
		} else {
			if score, ok := scores[node.Name]; ok && score < bestScore {
				bestNode = node
				bestScore = score
			}
		}
	}
	return bestNode
}

type EPVM struct {
	// Enhanced Parallel Virtual Machine
	Name string
}

/**
max_jobs = 1;
while () {
    machine_pick = 1; cost = MAX_COST
    repeat {} until (new job j arrives)
    for (each machine m) {
        marginal_cost = power(n, percentage memory utilization on
        ➥ m if j was added) +
        power(n, (jobs on m + 1/max_jobs) - power(n, memory use on m)
        ➥ - power(n, jobs on m/max_jobs));

        if (marginal_cost < cost) { machine_pick = m; }
    }

    assign job to machine_pick;
    if (jobs on machine_pick > max_jobs) max_jobs = max_jobs * 2;
}
*/

func (s *EPVM) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	var candidates []*node.Node
	for _, node := range nodes {
		diskAvailable := node.Disk - node.DiskAllocated
		if t.Disk <= diskAvailable {
			candidates = append(candidates, node)
		}
	}
	return candidates
}

func (s *EPVM) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	scores := make(map[string]float64)
	maxJobs := 4.0
	for _, node := range nodes {
		cpuUsage, err := calculateCpuUsage(node)
		if err != nil {
			continue
		}
		cpuLoad := *cpuUsage / math.Pow(2, 0.8)
		cpuCost := math.Pow(LIEB, cpuLoad) + math.Pow(LIEB, (float64(node.TaskCount)+1)/maxJobs) - math.Pow(LIEB, cpuLoad) - math.Pow(LIEB, float64(node.TaskCount)/maxJobs)

		memAllocated := float64(node.Stats.MemStat.Used + node.Stats.MemStat.Available)
		memAllocatedPercent := memAllocated / float64(node.Memory)
		memLoad := (memAllocated + float64(t.Memory)) / float64(node.Memory)
		memCost := math.Pow(LIEB, memLoad) + math.Pow(LIEB, (float64(node.TaskCount+1))/maxJobs) - math.Pow(LIEB, memAllocatedPercent) - math.Pow(LIEB, float64(node.TaskCount)/float64(maxJobs))

		scores[node.Name] = cpuCost + memCost
	}
	return scores
}

func (s *EPVM) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var bestScore float64
	for i, node := range candidates {
		if i == 0 {
			bestNode = node
			bestScore = scores[node.Name]
		} else {
			if score, ok := scores[node.Name]; ok && score < bestScore {
				bestNode = node
				bestScore = score
			}
		}
	}
	return bestNode
}

func calculateCpuUsage(node *node.Node) (*float64, error) {
	//curStats, err := node.GetStats()
	//if err != nil {
	//	return nil, err
	//}
	//return &curStats.CpuPercent, nil

	stats1, err := node.GetStats()
	if err != nil {
		return nil, err
	}
	time.Sleep(3 * time.Second)
	stats2, err := node.GetStats()
	if err != nil {
		return nil, err
	}
	idle1 := stats1.CpuStat.Idle + stats1.CpuStat.Iowait
	idle2 := stats2.CpuStat.Idle + stats2.CpuStat.Iowait
	nonIdle1 := stats1.CpuStat.User + stats1.CpuStat.System + stats1.CpuStat.Nice + stats1.CpuStat.Softirq + stats1.CpuStat.Irq + stats1.CpuStat.Steal
	nonIdle2 := stats2.CpuStat.User + stats2.CpuStat.System + stats2.CpuStat.Nice + stats2.CpuStat.Softirq + stats2.CpuStat.Irq + stats2.CpuStat.Steal
	total1 := idle1 + nonIdle1
	total2 := idle2 + nonIdle2
	tot := total2 - total1
	idle := idle2 - idle1
	var cpuUsagePercent float64
	if tot == 0 {
		cpuUsagePercent = 0.0
	} else {
		cpuUsagePercent = float64(tot-idle) / float64(tot)
	}
	return &cpuUsagePercent, nil
}
