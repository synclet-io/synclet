package pipelineworkflows

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

type env struct {
	ctx         context.Context
	store       *pipelinefake.Storage
	workspaceID uuid.UUID
	conns       map[string]uuid.UUID
	advancer    *pipelinejobs.AdvanceWorkflowRuns
}

func newEnv(t *testing.T, names ...string) *env {
	t.Helper()

	e := &env{ctx: context.Background(), store: pipelinefake.New(), workspaceID: uuid.New(), conns: map[string]uuid.UUID{}}
	e.advancer = pipelinejobs.NewAdvanceWorkflowRuns(e.store, pipelineservice.Config{MaxConcurrentJobs: 10}, nil)

	for _, name := range names {
		id := uuid.New()
		e.store.ConnectionsTable.Insert(&pipelineservice.Connection{
			ID: id, WorkspaceID: e.workspaceID, Name: name, Status: pipelineservice.ConnectionStatusActive, MaxAttempts: 1,
		})
		e.store.ConfigureStreams(id, "users")
		e.conns[name] = id
	}

	return e
}

// unconfigure removes the configured streams of the named connection.
func (e *env) unconfigure(t *testing.T, name string) {
	t.Helper()

	require.NoError(t, e.store.ConfiguredCatalogs().Delete(e.ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.Equals(e.conns[name]),
	}))
}

func (e *env) step(name string, after ...string) pipelineservice.WorkflowGraphStep {
	step := pipelineservice.WorkflowGraphStep{ConnectionID: e.conns[name]}
	for _, upstream := range after {
		step.AfterConnectionIDs = append(step.AfterConnectionIDs, e.conns[upstream])
	}

	return step
}

func (e *env) create(t *testing.T, steps ...pipelineservice.WorkflowGraphStep) *WorkflowDetail {
	t.Helper()

	schedule := "0 * * * *"
	detail, err := NewCreateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, CreateWorkflowParams{
		WorkspaceID: e.workspaceID, Name: "nightly", Schedule: &schedule, Steps: steps,
	})
	require.NoError(t, err)

	return detail
}

// start starts a run and advances it as the workflow_run.started consumer does.
func (e *env) start(t *testing.T, workflowID uuid.UUID) *WorkflowRunDetail {
	t.Helper()

	run, err := NewStartWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, StartWorkflowRunParams{WorkflowID: workflowID, WorkspaceID: e.workspaceID})
	require.NoError(t, err)
	require.NoError(t, e.advancer.Execute(e.ctx, run.Run.ID))

	return run
}

func (e *env) stepRunOf(runID uuid.UUID, name string) *pipelineservice.WorkflowStepRun {
	for _, stepRun := range e.store.WorkflowStepRunsTable.All() {
		if stepRun.RunID == runID && stepRun.ConnectionID == e.conns[name] {
			return stepRun
		}
	}

	return nil
}

func (e *env) finish(t *testing.T, runID uuid.UUID, name string, status pipelineservice.JobStatus) {
	t.Helper()

	stepRun := e.stepRunOf(runID, name)
	require.NotNil(t, stepRun.JobID, "step %s has no job", name)

	job := e.store.JobsTable.Get(*stepRun.JobID)
	job.Status = status
	now := time.Now()
	job.CompletedAt = &now
	_, err := e.store.Jobs().Update(e.ctx, job)
	require.NoError(t, err)
	require.NoError(t, e.advancer.ExecuteForJob(e.ctx, job))
}

func (e *env) jobsFor(name string) int {
	count := 0

	for _, job := range e.store.JobsTable.All() {
		if job.ConnectionID == e.conns[name] {
			count++
		}
	}

	return count
}

func requireValidationError(t *testing.T, err error, contains string) {
	t.Helper()

	var validationErr *pipelineservice.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Contains(t, err.Error(), contains)
}

func TestCreateWorkflow(t *testing.T) {
	e := newEnv(t, "A", "B", "C", "D")

	detail := e.create(t, e.step("A"), e.step("B", "A"), e.step("C", "A"), e.step("D", "B", "C"))

	assert.Equal(t, "nightly", detail.Workflow.Name)
	assert.Equal(t, e.workspaceID, detail.Workflow.WorkspaceID)
	require.NotNil(t, detail.Workflow.NextScheduledAt, "next_scheduled_at comes from the cron schedule")
	assert.True(t, detail.Workflow.NextScheduledAt.After(time.Now()))
	assert.Len(t, detail.Steps, 4)
	assert.Len(t, detail.Dependencies, 4)
	assert.Len(t, e.store.WorkflowsTable.All(), 1)

	got, err := NewGetWorkflow(e.store).Execute(e.ctx, GetWorkflowParams{ID: detail.Workflow.ID, WorkspaceID: e.workspaceID})
	require.NoError(t, err)
	assert.Len(t, got.Steps, 4)

	_, err = NewGetWorkflow(e.store).Execute(e.ctx, GetWorkflowParams{ID: detail.Workflow.ID, WorkspaceID: uuid.New()})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowNotFound, "another workspace does not see the workflow")
}

func TestCreateWorkflow_Rejects(t *testing.T) {
	e := newEnv(t, "A", "B")

	foreign := uuid.New()
	e.store.ConnectionsTable.Insert(&pipelineservice.Connection{ID: foreign, WorkspaceID: uuid.New(), Name: "other", Status: pipelineservice.ConnectionStatusActive})

	badCron := "every day"

	tests := []struct {
		name     string
		steps    []pipelineservice.WorkflowGraphStep
		schedule *string
		contains string
	}{
		{name: "cycle", steps: []pipelineservice.WorkflowGraphStep{e.step("A", "B"), e.step("B", "A")}, contains: "cycle"},
		{name: "connection listed twice", steps: []pipelineservice.WorkflowGraphStep{e.step("A"), e.step("A")}, contains: "more than once"},
		{name: "step from another workspace", steps: []pipelineservice.WorkflowGraphStep{e.step("A"), {ConnectionID: foreign}}, contains: "not a connection of this workspace"},
		{name: "invalid cron", steps: []pipelineservice.WorkflowGraphStep{e.step("A")}, schedule: &badCron, contains: "schedule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCreateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, CreateWorkflowParams{
				WorkspaceID: e.workspaceID, Name: "wf-" + tt.name, Schedule: tt.schedule, Steps: tt.steps,
			})
			requireValidationError(t, err, tt.contains)
			assert.Empty(t, e.store.WorkflowsTable.All(), "nothing is stored")
		})
	}
}

func TestCreateAndUpdateWorkflow_RejectConnectionWithoutStreams(t *testing.T) {
	e := newEnv(t, "A", "B", "C")
	e.unconfigure(t, "C")

	_, err := NewCreateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, CreateWorkflowParams{
		WorkspaceID: e.workspaceID, Name: "wf", Steps: []pipelineservice.WorkflowGraphStep{e.step("A"), e.step("C", "A")},
	})
	requireValidationError(t, err, `connection "C" has no configured streams`)
	assert.Empty(t, e.store.WorkflowsTable.All(), "nothing is stored")

	detail := e.create(t, e.step("A"), e.step("B", "A"))

	_, err = NewUpdateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, UpdateWorkflowParams{
		ID: detail.Workflow.ID, WorkspaceID: e.workspaceID, Name: "nightly",
		Steps: []pipelineservice.WorkflowGraphStep{e.step("A"), e.step("C", "A")},
	})
	requireValidationError(t, err, `connection "C" has no configured streams`)
	assert.Len(t, e.store.WorkflowStepsTable.All(), 2, "the steps stay as they were")
}

func TestRunWorkflow_RefusesConnectionWithoutStreams(t *testing.T) {
	e := newEnv(t, "A", "B")
	detail := e.create(t, e.step("A"), e.step("B", "A"))
	e.unconfigure(t, "B")

	_, err := NewStartWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, StartWorkflowRunParams{
		WorkflowID: detail.Workflow.ID, WorkspaceID: e.workspaceID,
	})

	var noStreams *pipelineservice.NoConfiguredStreamsError
	require.ErrorAs(t, err, &noStreams)
	assert.Equal(t, []string{"B"}, noStreams.ConnectionNames)
	assert.Empty(t, e.store.WorkflowRunsTable.All(), "no run is created")
	assert.Empty(t, e.store.JobsTable.All(), "no job is created")
}

func TestRerunWorkflowRun_RefusesConnectionWithoutStreams(t *testing.T) {
	e := newEnv(t, "A")
	detail := e.create(t, e.step("A"))
	runID := e.start(t, detail.Workflow.ID).Run.ID
	e.finish(t, runID, "A", pipelineservice.JobStatusFailed)
	e.unconfigure(t, "A")
	jobsBefore := e.jobsFor("A")

	_, err := NewRerunWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, RerunWorkflowRunParams{RunID: runID, WorkspaceID: e.workspaceID})

	var noStreams *pipelineservice.NoConfiguredStreamsError
	require.ErrorAs(t, err, &noStreams)
	assert.Equal(t, pipelineservice.WorkflowRunStatusFailed, e.store.WorkflowRunsTable.Get(runID).Status, "the run stays failed")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusFailed, e.stepRunOf(runID, "A").Status)
	assert.Equal(t, jobsBefore, e.jobsFor("A"), "no job is created")
}

func TestUpdateWorkflow(t *testing.T) {
	e := newEnv(t, "A", "B", "C")
	detail := e.create(t, e.step("A"), e.step("B", "A"))

	stepID := map[uuid.UUID]uuid.UUID{}
	for _, step := range detail.Steps {
		stepID[step.ConnectionID] = step.ID
	}

	updated, err := NewUpdateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, UpdateWorkflowParams{
		ID: detail.Workflow.ID, WorkspaceID: e.workspaceID, Name: "renamed",
		Steps: []pipelineservice.WorkflowGraphStep{e.step("A"), e.step("C", "A")},
	})
	require.NoError(t, err)

	assert.Equal(t, "renamed", updated.Workflow.Name)
	assert.Nil(t, updated.Workflow.NextScheduledAt, "no schedule, no scheduled start")
	require.Len(t, updated.Steps, 2)

	for _, step := range updated.Steps {
		if step.ConnectionID == e.conns["A"] {
			assert.Equal(t, stepID[e.conns["A"]], step.ID, "a connection that stays keeps its step id")
		}

		assert.NotEqual(t, e.conns["B"], step.ConnectionID, "B was removed")
	}

	require.Len(t, updated.Dependencies, 1)
	assert.Len(t, e.store.WorkflowStepsTable.All(), 2)
	assert.Len(t, e.store.WorkflowStepDependenciesTable.All(), 1)

	_, err = NewUpdateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, UpdateWorkflowParams{
		ID: detail.Workflow.ID, WorkspaceID: e.workspaceID, Name: "renamed",
		Steps: []pipelineservice.WorkflowGraphStep{e.step("A", "C"), e.step("C", "A")},
	})
	requireValidationError(t, err, "cycle")
}

func TestUpdateAndDeleteWorkflow_RejectedDuringActiveRun(t *testing.T) {
	e := newEnv(t, "A")
	detail := e.create(t, e.step("A"))
	e.start(t, detail.Workflow.ID)

	_, err := NewUpdateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, UpdateWorkflowParams{
		ID: detail.Workflow.ID, WorkspaceID: e.workspaceID, Name: "x", Steps: []pipelineservice.WorkflowGraphStep{e.step("A")},
	})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowRunActive)

	err = NewDeleteWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, DeleteWorkflowParams{ID: detail.Workflow.ID, WorkspaceID: e.workspaceID})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowRunActive)
	assert.Len(t, e.store.WorkflowsTable.All(), 1)
}

func TestDeleteWorkflow(t *testing.T) {
	e := newEnv(t, "A", "B")
	detail := e.create(t, e.step("A"), e.step("B", "A"))

	err := NewDeleteWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, DeleteWorkflowParams{ID: detail.Workflow.ID, WorkspaceID: uuid.New()})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowNotFound)

	require.NoError(t, NewDeleteWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, DeleteWorkflowParams{ID: detail.Workflow.ID, WorkspaceID: e.workspaceID}))
	assert.Empty(t, e.store.WorkflowsTable.All())
	assert.Empty(t, e.store.WorkflowStepsTable.All())
	assert.Empty(t, e.store.WorkflowStepDependenciesTable.All())
}

func TestStartWorkflowRun(t *testing.T) {
	e := newEnv(t, "A", "B")
	detail := e.create(t, e.step("A"), e.step("B", "A"))

	run := e.start(t, detail.Workflow.ID)
	assert.Equal(t, pipelineservice.WorkflowRunTriggerManual, run.Run.Trigger)
	assert.Equal(t, pipelineservice.WorkflowRunStatusRunning, run.Run.Status)
	require.Len(t, run.StepRuns, 2)
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, e.stepRunOf(run.Run.ID, "A").Status, "the first step is enqueued at once")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, e.stepRunOf(run.Run.ID, "B").Status)

	_, err := NewStartWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, StartWorkflowRunParams{WorkflowID: detail.Workflow.ID, WorkspaceID: e.workspaceID})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowRunActive, "a start while a run is active is rejected")

	_, err = NewStartWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, StartWorkflowRunParams{WorkflowID: detail.Workflow.ID, WorkspaceID: uuid.New()})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowNotFound)
}

func TestRerunWorkflowRun(t *testing.T) {
	// A → (B, C), B → D. B fails, so D is skipped; C succeeds.
	e := newEnv(t, "A", "B", "C", "D")
	detail := e.create(t, e.step("A"), e.step("B", "A"), e.step("C", "A"), e.step("D", "B"))
	runID := e.start(t, detail.Workflow.ID).Run.ID

	e.finish(t, runID, "A", pipelineservice.JobStatusCompleted)
	e.finish(t, runID, "B", pipelineservice.JobStatusFailed)
	e.finish(t, runID, "C", pipelineservice.JobStatusCompleted)
	require.Equal(t, pipelineservice.WorkflowRunStatusFailed, e.store.WorkflowRunsTable.Get(runID).Status)
	require.Equal(t, pipelineservice.WorkflowStepRunStatusSkipped, e.stepRunOf(runID, "D").Status)

	rerun := NewRerunWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{})

	run, err := rerun.Execute(e.ctx, RerunWorkflowRunParams{RunID: runID, WorkspaceID: e.workspaceID})
	require.NoError(t, err)
	require.NoError(t, e.advancer.Execute(e.ctx, runID))
	assert.Equal(t, runID, run.Run.ID, "the rerun reuses the failed run")
	assert.Equal(t, pipelineservice.WorkflowRunStatusRunning, run.Run.Status)
	assert.Nil(t, run.Run.CompletedAt)

	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSucceeded, e.stepRunOf(runID, "A").Status)
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusSucceeded, e.stepRunOf(runID, "C").Status)
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, e.stepRunOf(runID, "B").Status, "the failed step syncs again")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, e.stepRunOf(runID, "D").Status, "the skipped step waits for it")
	assert.Equal(t, 1, e.jobsFor("A"), "a step that succeeded is not synced again")
	assert.Equal(t, 1, e.jobsFor("C"), "a step that succeeded is not synced again")
	assert.Equal(t, 2, e.jobsFor("B"))

	_, err = rerun.Execute(e.ctx, RerunWorkflowRunParams{RunID: runID, WorkspaceID: e.workspaceID})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowRunNotFailed, "only a failed run can be rerun")

	e.finish(t, runID, "B", pipelineservice.JobStatusCompleted)
	e.finish(t, runID, "D", pipelineservice.JobStatusCompleted)
	assert.Equal(t, pipelineservice.WorkflowRunStatusSucceeded, e.store.WorkflowRunsTable.Get(runID).Status)
	assert.Equal(t, 1, e.jobsFor("A"))

	_, err = rerun.Execute(e.ctx, RerunWorkflowRunParams{RunID: runID, WorkspaceID: uuid.New()})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowRunNotFound, "another workspace cannot rerun the run")
}

func TestRerunWorkflowRun_RejectedWhileAnotherRunIsActive(t *testing.T) {
	e := newEnv(t, "A")
	detail := e.create(t, e.step("A"))
	runID := e.start(t, detail.Workflow.ID).Run.ID
	e.finish(t, runID, "A", pipelineservice.JobStatusFailed)

	e.start(t, detail.Workflow.ID)

	_, err := NewRerunWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, RerunWorkflowRunParams{RunID: runID, WorkspaceID: e.workspaceID})
	require.ErrorIs(t, err, pipelineservice.ErrWorkflowRunActive)
}

func (e *env) runStartedEvents() []*pipelineservice.WorkflowRunStartedEventData {
	var started []*pipelineservice.WorkflowRunStartedEventData

	for _, event := range e.store.Events {
		if data, ok := event.Data.(*pipelineservice.WorkflowRunStartedEventData); ok {
			started = append(started, data)
		}
	}

	return started
}

func TestStartWorkflowRun_EmitsEventAndCreatesNoJobs(t *testing.T) {
	e := newEnv(t, "A", "B")
	detail := e.create(t, e.step("A"), e.step("B", "A"))

	run, err := NewStartWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, StartWorkflowRunParams{WorkflowID: detail.Workflow.ID, WorkspaceID: e.workspaceID})
	require.NoError(t, err)

	assert.Empty(t, e.store.JobsTable.All(), "the API process enqueues no jobs")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, e.stepRunOf(run.Run.ID, "A").Status)
	assert.Equal(t, []*pipelineservice.WorkflowRunStartedEventData{{
		WorkspaceID: e.workspaceID, WorkflowID: detail.Workflow.ID, RunID: run.Run.ID,
	}}, e.runStartedEvents())
}

func TestRerunWorkflowRun_EmitsEventAndCreatesNoJobs(t *testing.T) {
	e := newEnv(t, "A")
	detail := e.create(t, e.step("A"))
	runID := e.start(t, detail.Workflow.ID).Run.ID
	e.finish(t, runID, "A", pipelineservice.JobStatusFailed)

	e.store.Events = nil
	jobsBefore := e.jobsFor("A")

	run, err := NewRerunWorkflowRun(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, RerunWorkflowRunParams{RunID: runID, WorkspaceID: e.workspaceID})
	require.NoError(t, err)

	assert.Equal(t, jobsBefore, e.jobsFor("A"), "the API process enqueues no jobs")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusPending, e.stepRunOf(runID, "A").Status)
	assert.Equal(t, []*pipelineservice.WorkflowRunStartedEventData{{
		WorkspaceID: e.workspaceID, WorkflowID: detail.Workflow.ID, RunID: run.Run.ID,
	}}, e.runStartedEvents())
}
