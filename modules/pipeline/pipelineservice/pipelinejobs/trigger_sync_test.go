package pipelinejobs

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

type triggerSyncStorage struct {
	pipelineservice.Storage
	conn *pipelineservice.Connection
}

func (s *triggerSyncStorage) Connections() pipelineservice.ConnectionsStorage {
	return &triggerSyncConnections{conn: s.conn}
}

func (s *triggerSyncStorage) Jobs() pipelineservice.JobsStorage { return &triggerSyncJobs{} }

func (s *triggerSyncStorage) PipelineEvents() pipelineservice.PipelineEventsOutbox {
	return &mockPipelineEventsOutbox{}
}

func (s *triggerSyncStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type triggerSyncConnections struct {
	pipelineservice.ConnectionsStorage
	conn *pipelineservice.Connection
}

func (c *triggerSyncConnections) First(_ context.Context, _ *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	return c.conn, nil
}

type triggerSyncJobs struct {
	pipelineservice.JobsStorage
}

func (j *triggerSyncJobs) Find(_ context.Context, _ *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Job, error) {
	return nil, nil
}

func (j *triggerSyncJobs) Create(_ context.Context, job *pipelineservice.Job) (*pipelineservice.Job, error) {
	return job, nil
}

type capturingAuditRecorder struct {
	events []pipelineservice.AuditEvent
}

func (r *capturingAuditRecorder) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}

func TestTriggerSync_RecordsAuditEvent(t *testing.T) {
	wsID := uuid.New()
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: wsID, Name: "pg-to-bq", MaxAttempts: 3}
	storage := &triggerSyncStorage{conn: conn}
	recorder := &capturingAuditRecorder{}

	uc := NewTriggerSync(
		pipelineconnections.NewGetConnection(storage),
		NewQueueJob(storage, pipelineservice.Config{}),
		recorder,
	)

	job, err := uc.Execute(context.Background(), TriggerSyncParams{ConnectionID: conn.ID, WorkspaceID: wsID})
	require.NoError(t, err)

	require.Len(t, recorder.events, 1)
	event := recorder.events[0]
	assert.Equal(t, wsID, event.WorkspaceID)
	assert.Equal(t, "sync", event.Action)
	assert.Equal(t, "connection", event.ResourceType)
	assert.Equal(t, conn.ID, event.ResourceID)
	assert.Equal(t, "pg-to-bq", event.ResourceLabel)
	assert.Equal(t, map[string]any{"job_id": job.ID}, event.After)
}
