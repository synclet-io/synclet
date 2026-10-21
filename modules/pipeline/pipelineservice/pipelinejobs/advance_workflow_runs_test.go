package pipelinejobs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// workflowFixture is a running workflow run whose steps are named by letter.
type workflowFixture struct {
	store    *pipelinefake.Storage
	runID    uuid.UUID
	steps    map[string]*pipelineservice.WorkflowStep
	stepRuns map[string]uuid.UUID
}

// newWorkflowFixture builds a workflow of the given steps, each with an
// active connection, and the edges as "step: upstream, upstream".
func newWorkflowFixture(t *testing.T, names []string, after map[string][]string) *workflowFixture {
	t.Helper()

	store := pipelinefake.New()
	workspaceID := uuid.New()
	workflow := &pipelineservice.Workflow{ID: uuid.New(), WorkspaceID: workspaceID, Name: "wf"}
	store.WorkflowsTable.Insert(workflow)

	run := &pipelineservice.WorkflowRun{
		ID: uuid.New(), WorkflowID: workflow.ID, Status: pipelineservice.WorkflowRunStatusRunning,
		Trigger: pipelineservice.WorkflowRunTriggerManual, StartedAt: time.Now(),
	}
	store.WorkflowRunsTable.Insert(run)

	fx := &workflowFixture{store: store, runID: run.ID, steps: map[string]*pipelineservice.WorkflowStep{}, stepRuns: map[string]uuid.UUID{}}

	for _, name := range names {
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: workspaceID, Name: name, Status: pipelineservice.ConnectionStatusActive, MaxAttempts: 2}
		store.ConnectionsTable.Insert(conn)

		step := &pipelineservice.WorkflowStep{ID: uuid.New(), WorkflowID: workflow.ID, ConnectionID: conn.ID}
		store.WorkflowStepsTable.Insert(step)
		fx.steps[name] = step

		stepRun := &pipelineservice.WorkflowStepRun{
			ID: uuid.New(), RunID: run.ID, StepID: step.ID, ConnectionID: conn.ID, Status: pipelineservice.WorkflowStepRunStatusPending,
		}
		store.WorkflowStepRunsTable.Insert(stepRun)
		fx.stepRuns[name] = stepRun.ID
	}

	for name, upstreams := range after {
		for _, upstream := range upstreams {
			store.WorkflowStepDependenciesTable.Insert(&pipelineservice.WorkflowStepDependency{
				ID: uuid.New(), WorkflowID: workflow.ID, StepID: fx.steps[name].ID, UpstreamStepID: fx.steps[upstream].ID,
			})
		}
	}

	return fx
}

func (fx *workflowFixture) stepRun(name string) *pipelineservice.WorkflowStepRun {
	return fx.store.WorkflowStepRunsTable.Get(fx.stepRuns[name])
}

func (fx *workflowFixture) status(name string) pipelineservice.WorkflowStepRunStatus {
	return fx.stepRun(name).Status
}

func (fx *workflowFixture) runStatus() pipelineservice.WorkflowRunStatus {
	return fx.store.WorkflowRunsTable.Get(fx.runID).Status
}

// finishJob moves the job of a queued step to a final status.
func (fx *workflowFixture) finishJob(t *testing.T, name string, status pipelineservice.JobStatus) *pipelineservice.Job {
	t.Helper()

	stepRun := fx.stepRun(name)
	require.NotNil(t, stepRun.JobID, "step %s has no job", name)

	job := fx.store.JobsTable.Get(*stepRun.JobID)
	job.Status = status
	now := time.Now()
	job.CompletedAt = &now

	if status == pipelineservice.JobStatusFailed {
		msg := "boom"
		job.Error = &msg
	}

	_, err := fx.store.Jobs().Update(context.Background(), job)
	require.NoError(t, err)

	return job
}

func (fx *workflowFixture) jobsFor(name string) []*pipelineservice.Job {
	var result []*pipelineservice.Job

	for _, job := range fx.store.JobsTable.All() {
		if job.ConnectionID == fx.steps[name].ConnectionID {
			result = append(result, job)
		}
	}

	return result
}

func newAdvancer(store pipelineservice.Storage, limit int) *AdvanceWorkflowRuns {
	return NewAdvanceWorkflowRuns(store, pipelineservice.Config{MaxConcurrentJobs: limit}, nil)
}

func TestAdvanceWorkflowRuns_FanOutFanIn(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B", "C", "D"}, map[string][]string{"B": {"A"}, "C": {"A"}, "D": {"B", "C"}})
	advancer := newAdvancer(fx.store, 10)

	require.NoError(t, advancer.Execute(ctx, fx.runID))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("A"))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, fx.status("B"))
	require.Len(t, fx.jobsFor("A"), 1)

	jobA := fx.jobsFor("A")[0]
	assert.Equal(t, pipelineservice.JobStatusScheduled, jobA.Status)
	assert.Equal(t, pipelineservice.JobTypeSync, jobA.JobType)
	assert.Equal(t, 1, jobA.Attempt)
	assert.Equal(t, 2, jobA.MaxAttempts)
	require.NotNil(t, jobA.WorkflowStepRunID)
	assert.Equal(t, fx.stepRuns["A"], *jobA.WorkflowStepRunID, "the job carries the step run it belongs to")
	assert.Equal(t, pipelineservice.PipelineEventTypeJobCreated, fx.store.Events[0].EventType)

	finished := fx.finishJob(t, "A", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, finished))

	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSucceeded, fx.status("A"))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("B"), "B starts once A succeeded")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("C"), "C starts together with B")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, fx.status("D"))

	finished = fx.finishJob(t, "B", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, finished))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, fx.status("D"), "D waits for C")
	assert.Empty(t, fx.jobsFor("D"))

	finished = fx.finishJob(t, "C", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, finished))
	require.NoError(t, advancer.Execute(ctx, fx.runID), "a second sweep enqueues nothing more")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("D"))
	assert.Len(t, fx.jobsFor("D"), 1, "D is enqueued exactly once")

	finished = fx.finishJob(t, "D", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, finished))
	assert.Equal(t, pipelineservice.WorkflowRunStatusSucceeded, fx.runStatus())
	assert.NotNil(t, fx.store.WorkflowRunsTable.Get(fx.runID).CompletedAt)
	assert.Contains(t, fx.store.Locks, "sync_scheduler", "the advancer runs under the scheduler lock")
}

func TestAdvanceWorkflowRuns_RetryKeepsStepQueued(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B"}, map[string][]string{"B": {"A"}})
	advancer := newAdvancer(fx.store, 10)

	require.NoError(t, advancer.Execute(ctx, fx.runID))

	retried := fx.finishJob(t, "A", pipelineservice.JobStatusScheduled)
	require.NoError(t, advancer.ExecuteForJob(ctx, retried))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("A"), "a job between attempts keeps its step queued")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, fx.status("B"))
}

func TestAdvanceWorkflowRuns_WaitsForActiveConnectionJob(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A"}, nil)
	advancer := newAdvancer(fx.store, 10)

	own := &pipelineservice.Job{
		ID: uuid.New(), ConnectionID: fx.steps["A"].ConnectionID, Status: pipelineservice.JobStatusRunning,
		JobType: pipelineservice.JobTypeSync, Attempt: 1, MaxAttempts: 1,
	}
	fx.store.JobsTable.Insert(own)

	require.NoError(t, advancer.Execute(ctx, fx.runID))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, fx.status("A"), "the step waits for the connection's own job")
	assert.Len(t, fx.jobsFor("A"), 1)

	own.Status = pipelineservice.JobStatusCompleted
	_, err := fx.store.Jobs().Update(ctx, own)
	require.NoError(t, err)

	require.NoError(t, advancer.ExecuteForJob(ctx, own), "the end of the connection's own job advances the waiting run")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("A"))
	assert.Len(t, fx.jobsFor("A"), 2, "the step enqueues its own job after the other one ended")
}

func TestAdvanceWorkflowRuns_RespectsConcurrencyCap(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B", "C"}, nil)

	busy := &pipelineservice.Job{ID: uuid.New(), ConnectionID: uuid.New(), Status: pipelineservice.JobStatusRunning, JobType: pipelineservice.JobTypeSync}
	fx.store.JobsTable.Insert(busy)

	advancer := newAdvancer(fx.store, 2)
	require.NoError(t, advancer.Execute(ctx, fx.runID))

	queued := 0

	for _, name := range []string{"A", "B", "C"} {
		if fx.status(name) == pipelineservice.WorkflowStepRunStatusQueued {
			queued++
		}
	}

	assert.Equal(t, 1, queued, "one slot is left under a cap of 2 with one job running")
	assert.Equal(t, pipelineservice.WorkflowRunStatusRunning, fx.runStatus())

	busy.Status = pipelineservice.JobStatusCompleted
	_, err := fx.store.Jobs().Update(ctx, busy)
	require.NoError(t, err)

	require.NoError(t, advancer.Execute(ctx, fx.runID))

	queued = 0

	for _, name := range []string{"A", "B", "C"} {
		if fx.status(name) == pipelineservice.WorkflowStepRunStatusQueued {
			queued++
		}
	}

	assert.Equal(t, 2, queued, "a freed slot goes to the next pending step")
}

func TestAdvanceWorkflowRuns_FailureSkipsDownstream(t *testing.T) {
	ctx := context.Background()
	// A → B → C, and an independent X → Y.
	fx := newWorkflowFixture(t, []string{"A", "B", "C", "X", "Y"}, map[string][]string{"B": {"A"}, "C": {"B"}, "Y": {"X"}})
	advancer := newAdvancer(fx.store, 10)

	require.NoError(t, advancer.Execute(ctx, fx.runID))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("A"))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("X"))

	failed := fx.finishJob(t, "A", pipelineservice.JobStatusFailed)
	require.NoError(t, advancer.ExecuteForJob(ctx, failed))

	assert.Equal(t, pipelineservice.WorkflowStepRunStatusFailed, fx.status("A"))
	require.NotNil(t, fx.stepRun("A").Error)
	assert.Equal(t, "boom", *fx.stepRun("A").Error, "the step keeps the job's error")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSkipped, fx.status("B"))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSkipped, fx.status("C"))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("X"), "the independent branch runs on")
	assert.Equal(t, pipelineservice.WorkflowRunStatusRunning, fx.runStatus())

	done := fx.finishJob(t, "X", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, done))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("Y"))

	done = fx.finishJob(t, "Y", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, done))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSucceeded, fx.status("Y"))
	assert.Equal(t, pipelineservice.WorkflowRunStatusFailed, fx.runStatus(), "a failed step fails the run")
	assert.Empty(t, fx.jobsFor("B"))
	assert.Empty(t, fx.jobsFor("C"))
}

func TestAdvanceWorkflowRuns_CancelledJobFailsStep(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B"}, map[string][]string{"B": {"A"}})
	advancer := newAdvancer(fx.store, 10)

	require.NoError(t, advancer.Execute(ctx, fx.runID))

	cancelled := fx.finishJob(t, "A", pipelineservice.JobStatusCancelled)
	require.NoError(t, advancer.ExecuteForJob(ctx, cancelled))

	assert.Equal(t, pipelineservice.WorkflowStepRunStatusFailed, fx.status("A"))
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSkipped, fx.status("B"))
	assert.Equal(t, pipelineservice.WorkflowRunStatusFailed, fx.runStatus())
}

func TestAdvanceWorkflowRuns_PausedConnectionSkipsStep(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B", "X"}, map[string][]string{"B": {"A"}})

	conn := fx.store.ConnectionsTable.Get(fx.steps["A"].ConnectionID)
	conn.Status = pipelineservice.ConnectionStatusPaused
	_, err := fx.store.Connections().Update(ctx, conn)
	require.NoError(t, err)

	advancer := newAdvancer(fx.store, 10)
	require.NoError(t, advancer.Execute(ctx, fx.runID))

	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSkipped, fx.status("A"), "a paused connection's step is skipped")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSkipped, fx.status("B"), "and so is everything downstream of it")
	assert.Empty(t, fx.jobsFor("A"))

	done := fx.finishJob(t, "X", pipelineservice.JobStatusCompleted)
	require.NoError(t, advancer.ExecuteForJob(ctx, done))
	assert.Equal(t, pipelineservice.WorkflowRunStatusSucceeded, fx.runStatus(), "a paused-connection skip alone does not fail the run")
}
