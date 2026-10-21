package pipelineworkflows

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

func (e *env) schedule(t *testing.T, name, cron string) {
	t.Helper()

	conn := e.store.ConnectionsTable.Get(e.conns[name])
	next := time.Now().Add(time.Hour)
	conn.Schedule = &cron
	conn.NextScheduledAt = &next
	_, err := e.store.Connections().Update(e.ctx, conn)
	require.NoError(t, err)
}

func TestWorkflowStepConnectionHandsItsScheduleToTheWorkflow(t *testing.T) {
	e := newEnv(t, "A", "B", "C")
	e.schedule(t, "A", "0 * * * *")
	e.schedule(t, "B", "*/5 * * * *")
	e.schedule(t, "C", "0 0 * * *")

	detail := e.create(t, e.step("A"), e.step("B", "A"))

	for _, name := range []string{"A", "B"} {
		conn := e.store.ConnectionsTable.Get(e.conns[name])
		assert.Nil(t, conn.Schedule, "%s is run by the workflow, not by a schedule of its own", name)
		assert.Nil(t, conn.NextScheduledAt, "%s has no scheduled sync", name)
	}

	assert.NotNil(t, e.store.ConnectionsTable.Get(e.conns["C"]).Schedule, "a connection outside the workflow keeps its schedule")

	_, err := NewUpdateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, UpdateWorkflowParams{
		ID: detail.Workflow.ID, WorkspaceID: e.workspaceID, Name: "nightly",
		Steps: []pipelineservice.WorkflowGraphStep{e.step("A"), e.step("B", "A"), e.step("C", "B")},
	})
	require.NoError(t, err)

	conn := e.store.ConnectionsTable.Get(e.conns["C"])
	assert.Nil(t, conn.Schedule, "adding C to the workflow turns its schedule off")
	assert.Nil(t, conn.NextScheduledAt)

	cron := "0 * * * *"
	scheduleUpdate := &cron
	_, err = pipelineconnections.NewUpdateConnection(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, pipelineconnections.UpdateConnectionParams{
		ID: e.conns["A"], WorkspaceID: e.workspaceID, Schedule: &scheduleUpdate,
	})
	requireValidationError(t, err, "workflow")
	assert.Nil(t, e.store.ConnectionsTable.Get(e.conns["A"]).Schedule, "a step connection does not get a schedule back")

	name := "renamed"
	var noSchedule *string
	_, err = pipelineconnections.NewUpdateConnection(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, pipelineconnections.UpdateConnectionParams{
		ID: e.conns["A"], WorkspaceID: e.workspaceID, Name: &name, Schedule: &noSchedule,
	})
	require.NoError(t, err, "a step connection's other settings stay editable")

	trigger := pipelinejobs.NewTriggerSync(
		pipelineconnections.NewGetConnection(e.store),
		pipelinejobs.NewQueueJob(e.store, pipelineservice.Config{MaxConcurrentJobs: 10}),
		pipelineservice.NoopAuditRecorder{},
	)
	job, err := trigger.Execute(e.ctx, pipelinejobs.TriggerSyncParams{ConnectionID: e.conns["A"], WorkspaceID: e.workspaceID})
	require.NoError(t, err, "Sync Now still works for a step connection")
	assert.Equal(t, e.conns["A"], job.ConnectionID)
}

func TestListConnectionWorkflows(t *testing.T) {
	e := newEnv(t, "A", "B", "C")
	first := e.create(t, e.step("A"), e.step("B", "A"))

	schedule := ""
	second, err := NewCreateWorkflow(e.store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, CreateWorkflowParams{
		WorkspaceID: e.workspaceID, Name: "hourly", Schedule: &schedule, Steps: []pipelineservice.WorkflowGraphStep{e.step("A")},
	})
	require.NoError(t, err)

	got, err := NewListConnectionWorkflows(e.store).Execute(e.ctx, ListConnectionWorkflowsParams{
		WorkspaceID:   e.workspaceID,
		ConnectionIDs: []uuid.UUID{e.conns["A"], e.conns["B"], e.conns["C"]},
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{first.Workflow.Name, second.Workflow.Name}, workflowNames(got[e.conns["A"]]))
	assert.Equal(t, []string{"nightly"}, workflowNames(got[e.conns["B"]]))
	assert.Empty(t, got[e.conns["C"]])
}

func workflowNames(workflows []*pipelineservice.Workflow) []string {
	names := make([]string, 0, len(workflows))
	for _, workflow := range workflows {
		names = append(names, workflow.Name)
	}

	return names
}
