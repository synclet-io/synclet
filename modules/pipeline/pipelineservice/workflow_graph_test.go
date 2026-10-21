package pipelineservice

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type diamond struct {
	a, b, c, d uuid.UUID
	deps       []*WorkflowStepDependency
}

func newDiamond() diamond {
	g := diamond{a: uuid.New(), b: uuid.New(), c: uuid.New(), d: uuid.New()}
	edge := func(step, upstream uuid.UUID) *WorkflowStepDependency {
		return &WorkflowStepDependency{ID: uuid.New(), StepID: step, UpstreamStepID: upstream}
	}
	g.deps = []*WorkflowStepDependency{edge(g.b, g.a), edge(g.c, g.a), edge(g.d, g.b), edge(g.d, g.c)}

	return g
}

func TestValidateWorkflowGraph(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()

	tests := []struct {
		name    string
		steps   []WorkflowGraphStep
		wantErr string
	}{
		{
			name:  "fan-out and fan-in",
			steps: []WorkflowGraphStep{{ConnectionID: a}, {ConnectionID: b, AfterConnectionIDs: []uuid.UUID{a}}, {ConnectionID: c, AfterConnectionIDs: []uuid.UUID{a, b}}},
		},
		{
			name:    "cycle A→B→A",
			steps:   []WorkflowGraphStep{{ConnectionID: a, AfterConnectionIDs: []uuid.UUID{b}}, {ConnectionID: b, AfterConnectionIDs: []uuid.UUID{a}}},
			wantErr: "cycle",
		},
		{
			name:    "longer cycle",
			steps:   []WorkflowGraphStep{{ConnectionID: a, AfterConnectionIDs: []uuid.UUID{c}}, {ConnectionID: b, AfterConnectionIDs: []uuid.UUID{a}}, {ConnectionID: c, AfterConnectionIDs: []uuid.UUID{b}}},
			wantErr: "cycle",
		},
		{
			name:    "the same connection twice",
			steps:   []WorkflowGraphStep{{ConnectionID: a}, {ConnectionID: a}},
			wantErr: "more than once",
		},
		{
			name:    "self edge",
			steps:   []WorkflowGraphStep{{ConnectionID: a, AfterConnectionIDs: []uuid.UUID{a}}},
			wantErr: "itself",
		},
		{
			name:    "unknown upstream",
			steps:   []WorkflowGraphStep{{ConnectionID: a, AfterConnectionIDs: []uuid.UUID{b}}},
			wantErr: "not a step",
		},
		{
			name:    "no steps",
			wantErr: "at least one step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowGraph(tt.steps)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			var validationErr *ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestReadySteps(t *testing.T) {
	g := newDiamond()

	statuses := func(a, b, c, d WorkflowStepRunStatus) map[uuid.UUID]WorkflowStepRunStatus {
		return map[uuid.UUID]WorkflowStepRunStatus{g.a: a, g.b: b, g.c: c, g.d: d}
	}

	pending := WorkflowStepRunStatusPending
	queued := WorkflowStepRunStatusQueued
	succeeded := WorkflowStepRunStatusSucceeded

	assert.ElementsMatch(t, []uuid.UUID{g.a}, ReadySteps(statuses(pending, pending, pending, pending), g.deps), "only the root is ready at the start")
	assert.ElementsMatch(t, []uuid.UUID{g.b, g.c}, ReadySteps(statuses(succeeded, pending, pending, pending), g.deps), "A succeeded: B and C")
	assert.Empty(t, ReadySteps(statuses(succeeded, succeeded, queued, pending), g.deps), "B succeeded, C queued: D waits")
	assert.ElementsMatch(t, []uuid.UUID{g.d}, ReadySteps(statuses(succeeded, succeeded, succeeded, pending), g.deps), "B and C succeeded: D")
	assert.Empty(t, ReadySteps(statuses(succeeded, succeeded, succeeded, queued), g.deps), "a queued step is not ready again")
	assert.Empty(t, ReadySteps(statuses(succeeded, WorkflowStepRunStatusFailed, succeeded, pending), g.deps), "a failed upstream never makes D ready")
}

func TestDownstream(t *testing.T) {
	g := newDiamond()

	assert.ElementsMatch(t, []uuid.UUID{g.b, g.c, g.d}, Downstream(g.a, g.deps))
	assert.ElementsMatch(t, []uuid.UUID{g.d}, Downstream(g.b, g.deps))
	assert.Empty(t, Downstream(g.d, g.deps))
}

func TestRunOutcome(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	of := func(a, b WorkflowStepRunStatus) map[uuid.UUID]WorkflowStepRunStatus {
		return map[uuid.UUID]WorkflowStepRunStatus{ids[0]: a, ids[1]: b}
	}

	assert.Equal(t, WorkflowRunStatusRunning, RunOutcome(of(WorkflowStepRunStatusSucceeded, WorkflowStepRunStatusQueued)))
	assert.Equal(t, WorkflowRunStatusRunning, RunOutcome(of(WorkflowStepRunStatusFailed, WorkflowStepRunStatusPending)))
	assert.Equal(t, WorkflowRunStatusFailed, RunOutcome(of(WorkflowStepRunStatusFailed, WorkflowStepRunStatusSkipped)))
	assert.Equal(t, WorkflowRunStatusSucceeded, RunOutcome(of(WorkflowStepRunStatusSucceeded, WorkflowStepRunStatusSucceeded)))
	assert.Equal(t, WorkflowRunStatusSucceeded, RunOutcome(of(WorkflowStepRunStatusSucceeded, WorkflowStepRunStatusSkipped)), "a paused-connection skip alone does not fail the run")
}
