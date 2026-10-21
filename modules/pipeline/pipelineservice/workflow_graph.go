package pipelineservice

import (
	"slices"

	"github.com/google/uuid"
)

// WorkflowGraphStep is one step of a workflow definition, named by its
// connection, with the connections of the steps it runs after.
type WorkflowGraphStep struct {
	ConnectionID       uuid.UUID
	AfterConnectionIDs []uuid.UUID
}

// ValidateWorkflowGraph checks that a workflow definition has at least one
// step, lists each connection once, names only its own steps as upstreams,
// has no step after itself and no cycle. Every failure is a *ValidationError.
func ValidateWorkflowGraph(steps []WorkflowGraphStep) error {
	if len(steps) == 0 {
		return &ValidationError{Field: "steps", Message: "a workflow needs at least one step"}
	}

	known := make(map[uuid.UUID]bool, len(steps))
	for _, step := range steps {
		if known[step.ConnectionID] {
			return &ValidationError{Field: "steps", Message: "connection " + step.ConnectionID.String() + " is listed more than once"}
		}

		known[step.ConnectionID] = true
	}

	inDegree := make(map[uuid.UUID]int, len(steps))
	downstream := make(map[uuid.UUID][]uuid.UUID, len(steps))

	for _, step := range steps {
		for _, upstream := range uniqueIDs(step.AfterConnectionIDs) {
			if upstream == step.ConnectionID {
				return &ValidationError{Field: "steps", Message: "step " + step.ConnectionID.String() + " runs after itself"}
			}

			if !known[upstream] {
				return &ValidationError{Field: "steps", Message: "upstream " + upstream.String() + " of step " + step.ConnectionID.String() + " is not a step of this workflow"}
			}

			inDegree[step.ConnectionID]++
			downstream[upstream] = append(downstream[upstream], step.ConnectionID)
		}
	}

	queue := make([]uuid.UUID, 0, len(steps))
	for _, step := range steps {
		if inDegree[step.ConnectionID] == 0 {
			queue = append(queue, step.ConnectionID)
		}
	}

	visited := 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, next := range downstream[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if visited != len(steps) {
		return &ValidationError{Field: "steps", Message: "the steps form a cycle"}
	}

	return nil
}

// ReadySteps returns the pending steps whose upstream steps have all
// succeeded, ordered by id.
func ReadySteps(statuses map[uuid.UUID]WorkflowStepRunStatus, deps []*WorkflowStepDependency) []uuid.UUID {
	upstreams := make(map[uuid.UUID][]uuid.UUID)
	for _, dep := range deps {
		upstreams[dep.StepID] = append(upstreams[dep.StepID], dep.UpstreamStepID)
	}

	var ready []uuid.UUID

	for stepID, status := range statuses {
		if status != WorkflowStepRunStatusPending {
			continue
		}

		allSucceeded := true

		for _, upstream := range upstreams[stepID] {
			if statuses[upstream] != WorkflowStepRunStatusSucceeded {
				allSucceeded = false

				break
			}
		}

		if allSucceeded {
			ready = append(ready, stepID)
		}
	}

	sortIDs(ready)

	return ready
}

// Downstream returns every step that runs after stepID, directly or through
// other steps, ordered by id.
func Downstream(stepID uuid.UUID, deps []*WorkflowStepDependency) []uuid.UUID {
	children := make(map[uuid.UUID][]uuid.UUID)
	for _, dep := range deps {
		children[dep.UpstreamStepID] = append(children[dep.UpstreamStepID], dep.StepID)
	}

	seen := map[uuid.UUID]bool{stepID: true}
	queue := []uuid.UUID{stepID}

	var result []uuid.UUID

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, child := range children[current] {
			if seen[child] {
				continue
			}

			seen[child] = true
			result = append(result, child)
			queue = append(queue, child)
		}
	}

	sortIDs(result)

	return result
}

// RunOutcome is running while any step is pending or queued; then failed if
// any step failed, otherwise succeeded.
func RunOutcome(statuses map[uuid.UUID]WorkflowStepRunStatus) WorkflowRunStatus {
	failed := false

	for _, status := range statuses {
		switch status {
		case WorkflowStepRunStatusPending, WorkflowStepRunStatusQueued:
			return WorkflowRunStatusRunning
		case WorkflowStepRunStatusFailed:
			failed = true
		case WorkflowStepRunStatusSucceeded, WorkflowStepRunStatusSkipped:
		}
	}

	if failed {
		return WorkflowRunStatusFailed
	}

	return WorkflowRunStatusSucceeded
}

func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	result := make([]uuid.UUID, 0, len(ids))

	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

func sortIDs(ids []uuid.UUID) {
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
}
