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
	conn         *pipelineservice.Connection
	unconfigured bool
	jobs         triggerSyncJobs
}

func (s *triggerSyncStorage) Connections() pipelineservice.ConnectionsStorage {
	return &triggerSyncConnections{conn: s.conn, configured: !s.unconfigured}
}

func (s *triggerSyncStorage) Jobs() pipelineservice.JobsStorage { return &s.jobs }

func (s *triggerSyncStorage) PipelineEvents() pipelineservice.PipelineEventsOutbox {
	return &mockPipelineEventsOutbox{}
}

func (s *triggerSyncStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type triggerSyncConnections struct {
	pipelineservice.ConnectionsStorage
	conn       *pipelineservice.Connection
	configured bool
}

func (c *triggerSyncConnections) WithConfiguredStreams(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	configured := map[uuid.UUID]bool{}
	for _, id := range ids {
		configured[id] = c.configured
	}

	return configured, nil
}

func (c *triggerSyncConnections) First(_ context.Context, _ *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	return c.conn, nil
}

type triggerSyncJobs struct {
	pipelineservice.JobsStorage
	created []*pipelineservice.Job
}

func (j *triggerSyncJobs) Find(_ context.Context, _ *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Job, error) {
	return nil, nil
}

func (j *triggerSyncJobs) Create(_ context.Context, job *pipelineservice.Job) (*pipelineservice.Job, error) {
	j.created = append(j.created, job)

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

func TestTriggerSync_RefusesConnectionWithoutStreams(t *testing.T) {
	wsID := uuid.New()
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: wsID, Name: "pg-to-bq", MaxAttempts: 3}
	storage := &triggerSyncStorage{conn: conn, unconfigured: true}
	recorder := &capturingAuditRecorder{}

	uc := NewTriggerSync(
		pipelineconnections.NewGetConnection(storage),
		NewQueueJob(storage, pipelineservice.Config{}),
		recorder,
	)

	_, err := uc.Execute(context.Background(), TriggerSyncParams{ConnectionID: conn.ID, WorkspaceID: wsID})

	var noStreams *pipelineservice.NoConfiguredStreamsError
	require.ErrorAs(t, err, &noStreams)
	assert.Equal(t, []string{"pg-to-bq"}, noStreams.ConnectionNames)
	assert.Contains(t, err.Error(), `connection "pg-to-bq" has no configured streams`)
	assert.Empty(t, storage.jobs.created, "no job is created")
	assert.Empty(t, recorder.events)
}
