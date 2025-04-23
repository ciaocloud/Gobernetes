package scheduler

import (
	"Gobernetes/node"
	"Gobernetes/task"
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
