package task

type State int

const (
	Pending   State = iota // The initial state (starting point) of a task
	Scheduled              // The manager has scheduled the task onto a worker
	Running                // The worker successfully started the task, i.e., the container is running
	Completed              // The task has completed without errors/failures
	Failed                 // The task has failed (either in scheduling, starting, running, or terminating)
)

var stateTransitionMap = map[State][]State{
	Pending:   []State{Scheduled}, // on event ScheduleEvent
	Scheduled: []State{Running, Failed, Scheduled},
	Running:   []State{Completed, Failed, Running}, // on event StopEvent
	Completed: []State{},
	Failed:    []State{},
}

func ValidateStateTransition(src State, dst State) bool {
	validNextStates := stateTransitionMap[src]
	for _, validState := range validNextStates {
		if validState == dst {
			return true
		}
	}
	return false
}
