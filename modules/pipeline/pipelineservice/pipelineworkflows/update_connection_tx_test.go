package pipelineworkflows

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// recordingStorage logs each connection and workflow step access, with
// whether it ran inside ExecuteInTransaction and whether the read locked rows.
type recordingStorage struct {
	pipelineservice.Storage
	inTx bool
	log  *[]string
}

func (s *recordingStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return s.Storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		return cb(ctx, &recordingStorage{Storage: tx, inTx: true, log: s.log})
	})
}

func (s *recordingStorage) Connections() pipelineservice.ConnectionsStorage {
	return &recordingConnections{ConnectionsStorage: s.Storage.Connections(), s: s}
}

func (s *recordingStorage) WorkflowSteps() pipelineservice.WorkflowStepsStorage {
	return &recordingSteps{WorkflowStepsStorage: s.Storage.WorkflowSteps(), s: s}
}

func (s *recordingStorage) record(format string, args ...any) {
	*s.log = append(*s.log, fmt.Sprintf(format, args...)+fmt.Sprintf(" tx=%v", s.inTx))
}

type recordingConnections struct {
	pipelineservice.ConnectionsStorage
	s *recordingStorage
}

func forUpdate(opts []optionutil.Option[dbutil.SelectOptions]) bool {
	return optionutil.ApplyOptions(&dbutil.SelectOptions{}, opts...).ForUpdate
}

func (c *recordingConnections) First(ctx context.Context, f *pipelineservice.ConnectionFilter, opts ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	c.s.record("read connection forUpdate=%v", forUpdate(opts))

	return c.ConnectionsStorage.First(ctx, f, opts...)
}

func (c *recordingConnections) Find(ctx context.Context, f *pipelineservice.ConnectionFilter, opts ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	c.s.record("read connections forUpdate=%v", forUpdate(opts))

	return c.ConnectionsStorage.Find(ctx, f, opts...)
}

func (c *recordingConnections) Update(ctx context.Context, conn *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	c.s.record("write connection")

	return c.ConnectionsStorage.Update(ctx, conn)
}

type recordingSteps struct {
	pipelineservice.WorkflowStepsStorage
	s *recordingStorage
}

func (r *recordingSteps) Count(ctx context.Context, f *pipelineservice.WorkflowStepFilter) (int, error) {
	r.s.record("count steps")

	return r.WorkflowStepsStorage.Count(ctx, f)
}

func TestUpdateConnection_ReadsChecksAndWritesUnderOneRowLock(t *testing.T) {
	e := newEnv(t, "A")
	e.create(t, e.step("A"))

	var log []string
	store := &recordingStorage{Storage: e.store, log: &log}

	cron := "0 * * * *"
	schedule := &cron
	_, err := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, pipelineconnections.UpdateConnectionParams{
		ID: e.conns["A"], WorkspaceID: e.workspaceID, Schedule: &schedule,
	})
	requireValidationError(t, err, "workflow")

	assert.Equal(t, []string{
		"read connection forUpdate=true tx=true",
		"count steps tx=true",
	}, log)

	log = nil
	name := "renamed"
	_, err = pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, pipelineconnections.UpdateConnectionParams{
		ID: e.conns["A"], WorkspaceID: e.workspaceID, Name: &name,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"read connection forUpdate=true tx=true",
		"write connection tx=true",
	}, log)
}

func TestWorkflowHandoverLocksStepConnections(t *testing.T) {
	e := newEnv(t, "A", "B")
	e.schedule(t, "A", "0 * * * *")

	var log []string
	store := &recordingStorage{Storage: e.store, log: &log}

	detail, err := NewCreateWorkflow(store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, CreateWorkflowParams{
		WorkspaceID: e.workspaceID, Name: "nightly", Steps: []pipelineservice.WorkflowGraphStep{e.step("A")},
	})
	require.NoError(t, err)
	assert.Contains(t, log, "read connections forUpdate=true tx=true", "create locks the step connections before clearing their schedules")
	assert.Contains(t, log, "write connection tx=true")

	log = nil

	e.schedule(t, "B", "0 * * * *")
	_, err = NewUpdateWorkflow(store, pipelineservice.NoopAuditRecorder{}).Execute(e.ctx, UpdateWorkflowParams{
		ID: detail.Workflow.ID, WorkspaceID: e.workspaceID, Name: "nightly",
		Steps: []pipelineservice.WorkflowGraphStep{e.step("A"), e.step("B", "A")},
	})
	require.NoError(t, err)
	assert.Contains(t, log, "read connections forUpdate=true tx=true", "update locks the step connections before clearing their schedules")
	assert.Nil(t, e.store.ConnectionsTable.Get(e.conns["B"]).Schedule)
}
