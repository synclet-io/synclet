package pipelinejobs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func dueWorkflow(fx *workflowFixture) *pipelineservice.Workflow {
	var workflow *pipelineservice.Workflow
	for _, wf := range fx.store.WorkflowsTable.All() {
		workflow = wf
	}

	schedule := "*/5 * * * *"
	past := time.Now().Add(-time.Minute)
	workflow.Schedule = &schedule
	workflow.NextScheduledAt = &past
	_, _ = fx.store.Workflows().Update(context.Background(), workflow)

	return workflow
}

func runsOf(fx *workflowFixture, workflowID uuid.UUID) []*pipelineservice.WorkflowRun {
	var runs []*pipelineservice.WorkflowRun

	for _, run := range fx.store.WorkflowRunsTable.All() {
		if run.WorkflowID == workflowID {
			runs = append(runs, run)
		}
	}

	return runs
}

func TestSyncScheduler_StartsDueWorkflow(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B"}, map[string][]string{"B": {"A"}})
	workflow := dueWorkflow(fx)

	// The fixture's run is over; a scheduled start may begin a new one.
	finished := fx.store.WorkflowRunsTable.Get(fx.runID)
	finished.Status = pipelineservice.WorkflowRunStatusSucceeded
	_, err := fx.store.WorkflowRuns().Update(ctx, finished)
	require.NoError(t, err)

	scheduler := NewSyncScheduler(fx.store, pipelineservice.Config{MaxConcurrentJobs: 10}, nil)
	require.NoError(t, scheduler.Execute(ctx))

	runs := runsOf(fx, workflow.ID)
	require.Len(t, runs, 2)

	var started *pipelineservice.WorkflowRun

	for _, run := range runs {
		if run.ID != fx.runID {
			started = run
		}
	}

	assert.Equal(t, pipelineservice.WorkflowRunTriggerSchedule, started.Trigger)
	assert.Equal(t, pipelineservice.WorkflowRunStatusRunning, started.Status)
	assert.Len(t, fx.jobsFor("A"), 1, "the new run enqueues its first step")
	assert.Empty(t, fx.jobsFor("B"))

	next := fx.store.WorkflowsTable.Get(workflow.ID).NextScheduledAt
	require.NotNil(t, next)
	assert.True(t, next.After(time.Now()), "next_scheduled_at moves to the next cron tick")
}

func TestSyncScheduler_SkipsScheduledStartDuringActiveRun(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A"}, nil)
	workflow := dueWorkflow(fx)

	scheduler := NewSyncScheduler(fx.store, pipelineservice.Config{MaxConcurrentJobs: 10}, nil)
	require.NoError(t, scheduler.Execute(ctx))

	assert.Len(t, runsOf(fx, workflow.ID), 1, "no second run while one is active")

	next := fx.store.WorkflowsTable.Get(workflow.ID).NextScheduledAt
	require.NotNil(t, next)
	assert.True(t, next.After(time.Now()), "the skipped start still advances next_scheduled_at")
	assert.Equal(t, pipelineservice.WorkflowStepRunStatusQueued, fx.status("A"), "the sweep advances the active run")
}

func TestSyncScheduler_SweepRespectsCap(t *testing.T) {
	ctx := context.Background()
	fx := newWorkflowFixture(t, []string{"A", "B"}, nil)

	scheduler := NewSyncScheduler(fx.store, pipelineservice.Config{MaxConcurrentJobs: 1}, nil)
	require.NoError(t, scheduler.Execute(ctx))
	assert.Len(t, fx.store.JobsTable.All(), 1)

	require.NoError(t, scheduler.Execute(ctx))
	assert.Len(t, fx.store.JobsTable.All(), 1, "no job beyond the cap")
}
