package pipelinejobs

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// jobStateStorage holds a single job and applies conditional updates the way
// the Postgres storage does: a write only lands while the stored status matches.
type jobStateStorage struct {
	pipelineservice.Storage
	job      *pipelineservice.Job
	readJob  *pipelineservice.Job
	conn     *pipelineservice.Connection
	attempts []*pipelineservice.JobAttempt
	events   []*pipelineservice.PipelineEvent
}

func newJobStateStorage(status pipelineservice.JobStatus) *jobStateStorage {
	started := time.Now().Add(-time.Minute)
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New(), SourceID: uuid.New(), DestinationID: uuid.New()}

	return &jobStateStorage{
		conn: conn,
		job: &pipelineservice.Job{
			ID:           uuid.New(),
			ConnectionID: conn.ID,
			Status:       status,
			Attempt:      1,
			MaxAttempts:  3,
			StartedAt:    &started,
		},
	}
}

func (s *jobStateStorage) Jobs() pipelineservice.JobsStorage { return &jobStateJobs{s: s} }

func (s *jobStateStorage) JobAttempts() pipelineservice.JobAttemptsStorage {
	return &jobStateAttempts{s: s}
}

func (s *jobStateStorage) Connections() pipelineservice.ConnectionsStorage {
	return &jobStateConnections{s: s}
}

func (s *jobStateStorage) Sources() pipelineservice.SourcesStorage { return &jobStateSources{} }

func (s *jobStateStorage) PipelineEvents() pipelineservice.PipelineEventsOutbox {
	return &jobStateOutbox{s: s}
}

func (s *jobStateStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type jobStateJobs struct {
	pipelineservice.JobsStorage
	s *jobStateStorage
}

func (j *jobStateJobs) First(_ context.Context, _ *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Job, error) {
	src := j.s.job
	if j.s.readJob != nil {
		src = j.s.readJob
	}

	cp := *src

	return &cp, nil
}

func (j *jobStateJobs) ClaimNextScheduledJob(_ context.Context, workerID string) (*pipelineservice.Job, error) {
	j.s.job.Status = pipelineservice.JobStatusStarting
	j.s.job.WorkerID = &workerID
	cp := *j.s.job

	return &cp, nil
}

func (j *jobStateJobs) UpdateIfStatus(_ context.Context, job *pipelineservice.Job, fromStatuses ...pipelineservice.JobStatus) (bool, error) {
	if !slices.Contains(fromStatuses, j.s.job.Status) {
		return false, nil
	}

	cp := *job
	j.s.job = &cp

	return true, nil
}

func (j *jobStateJobs) MarkRunning(_ context.Context, _ uuid.UUID) (bool, error) {
	if j.s.job.Status != pipelineservice.JobStatusStarting {
		return false, nil
	}

	j.s.job.Status = pipelineservice.JobStatusRunning

	return true, nil
}

func (j *jobStateJobs) TouchHeartbeat(_ context.Context, _ uuid.UUID, at time.Time) (bool, error) {
	if !isJobInFlight(j.s.job.Status) {
		return false, nil
	}

	j.s.job.HeartbeatAt = &at

	return true, nil
}

type jobStateAttempts struct {
	pipelineservice.JobAttemptsStorage
	s *jobStateStorage
}

func (a *jobStateAttempts) Create(_ context.Context, attempt *pipelineservice.JobAttempt) (*pipelineservice.JobAttempt, error) {
	a.s.attempts = append(a.s.attempts, attempt)

	return attempt, nil
}

type jobStateConnections struct {
	pipelineservice.ConnectionsStorage
	s *jobStateStorage
}

func (c *jobStateConnections) First(_ context.Context, _ *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	return c.s.conn, nil
}

func (c *jobStateConnections) Update(_ context.Context, conn *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	return conn, nil
}

type jobStateSources struct {
	pipelineservice.SourcesStorage
}

func (s *jobStateSources) First(_ context.Context, _ *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	return nil, pipelineservice.ErrSourceNotFound
}

type jobStateOutbox struct {
	pipelineservice.PipelineEventsOutbox
	s *jobStateStorage
}

func (o *jobStateOutbox) Send(_ context.Context, event *pipelineservice.PipelineEvent) error {
	o.s.events = append(o.s.events, event)

	return nil
}

func jobUpdatedData(t *testing.T, event *pipelineservice.PipelineEvent) *pipelineservice.JobUpdatedEventData {
	t.Helper()

	require.Equal(t, pipelineservice.PipelineEventTypeJobUpdated, event.EventType)
	data, ok := event.Data.(*pipelineservice.JobUpdatedEventData)
	require.True(t, ok)

	return data
}

func TestUpdateJobStatus_CompletesRunningJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)

	err := NewUpdateJobStatus(storage).Execute(context.Background(), UpdateJobStatusParams{ID: storage.job.ID})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusCompleted, storage.job.Status)
	assert.Len(t, storage.attempts, 1)
	require.Len(t, storage.events, 1)

	data := jobUpdatedData(t, storage.events[0])
	assert.Equal(t, storage.conn.WorkspaceID, data.WorkspaceID)
	assert.Equal(t, pipelineservice.JobStatusRunning, data.OldData.Status)
	assert.Equal(t, pipelineservice.JobStatusCompleted, data.NewData.Status)
}

func TestUpdateJobStatus_TerminalJobIsUnchanged(t *testing.T) {
	for _, status := range []pipelineservice.JobStatus{
		pipelineservice.JobStatusCancelled,
		pipelineservice.JobStatusCompleted,
		pipelineservice.JobStatusFailed,
	} {
		t.Run(status.String(), func(t *testing.T) {
			storage := newJobStateStorage(status)

			err := NewUpdateJobStatus(storage).Execute(context.Background(), UpdateJobStatusParams{
				ID:      storage.job.ID,
				SyncErr: errors.New("connector exited"),
			})
			require.NoError(t, err)

			assert.Equal(t, status, storage.job.Status)
			assert.Empty(t, storage.attempts)
			assert.Empty(t, storage.events)
		})
	}
}

func TestUpdateJobStatus_CancelledAfterReadIsNotRetried(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusCancelled)
	running := *storage.job
	running.Status = pipelineservice.JobStatusRunning
	storage.readJob = &running

	err := NewUpdateJobStatus(storage).Execute(context.Background(), UpdateJobStatusParams{
		ID:      storage.job.ID,
		SyncErr: errors.New("connector exited"),
	})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusCancelled, storage.job.Status)
	assert.Equal(t, 1, storage.job.Attempt)
	assert.Empty(t, storage.attempts)
	assert.Empty(t, storage.events)
}

func TestUpdateJobStatus_SkipRetryFailsJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusStarting)

	err := NewUpdateJobStatus(storage).Execute(context.Background(), UpdateJobStatusParams{
		ID:        storage.job.ID,
		SyncErr:   errors.New("source deleted"),
		SkipRetry: true,
	})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusFailed, storage.job.Status)
	require.Len(t, storage.events, 1)
	assert.Equal(t, pipelineservice.JobStatusFailed, jobUpdatedData(t, storage.events[0]).NewData.Status)
}

func TestUpdateHeartbeat_DoesNotReviveCancelledJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusCancelled)

	err := NewUpdateHeartbeat(storage).Execute(context.Background(), UpdateHeartbeatParams{ID: storage.job.ID})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusCancelled, storage.job.Status)
	assert.Nil(t, storage.job.HeartbeatAt)
}

func TestUpdateHeartbeat_TouchesRunningJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)

	err := NewUpdateHeartbeat(storage).Execute(context.Background(), UpdateHeartbeatParams{ID: storage.job.ID})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusRunning, storage.job.Status)
	assert.NotNil(t, storage.job.HeartbeatAt)
}

func TestCancelJob_FinishedAfterReadIsRejected(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusCompleted)
	running := *storage.job
	running.Status = pipelineservice.JobStatusRunning
	storage.readJob = &running

	err := NewCancelJob(storage, nil).Execute(context.Background(), CancelJobParams{ID: storage.job.ID})

	var validationErr *pipelineservice.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, pipelineservice.JobStatusCompleted, storage.job.Status)
	assert.Empty(t, storage.events)
}

func TestCancelJob_CancelsRunningJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)

	err := NewCancelJob(storage, nil).Execute(context.Background(), CancelJobParams{ID: storage.job.ID})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusCancelled, storage.job.Status)
	require.Len(t, storage.events, 1)
	assert.Equal(t, pipelineservice.JobStatusCancelled, jobUpdatedData(t, storage.events[0]).NewData.Status)
}

func TestClaimJobBundle_MissingSourceFailsJobWithoutRetry(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusScheduled)

	uc := NewClaimJobBundle(NewClaimJob(storage, nil), NewUpdateJobStatus(storage), nil, nil, nil, storage, nil, nil)

	result, err := uc.Execute(context.Background(), ClaimJobParams{WorkerID: "worker-1"})
	require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	assert.Nil(t, result)

	assert.Equal(t, pipelineservice.JobStatusFailed, storage.job.Status)
	require.NotNil(t, storage.job.Error)
	assert.Contains(t, *storage.job.Error, "Source")
	assert.Len(t, storage.attempts, 1)
}

func (j *jobStateJobs) Find(_ context.Context, _ *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Job, error) {
	if !isJobInFlight(j.s.job.Status) {
		return nil, nil
	}

	cp := *j.s.job

	return []*pipelineservice.Job{&cp}, nil
}

func TestMarkJobRunning_MovesStartingJobToRunning(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusStarting)

	err := NewMarkJobRunning(storage).Execute(context.Background(), MarkJobRunningParams{ID: storage.job.ID})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusRunning, storage.job.Status)
	require.Len(t, storage.events, 1)

	data := jobUpdatedData(t, storage.events[0])
	assert.Equal(t, storage.conn.WorkspaceID, data.WorkspaceID)
	assert.Equal(t, pipelineservice.JobStatusStarting, data.OldData.Status)
	assert.Equal(t, pipelineservice.JobStatusRunning, data.NewData.Status)
}

func TestMarkJobRunning_LeavesOtherStatusesUnchanged(t *testing.T) {
	for _, status := range []pipelineservice.JobStatus{
		pipelineservice.JobStatusScheduled,
		pipelineservice.JobStatusRunning,
		pipelineservice.JobStatusCancelled,
		pipelineservice.JobStatusCompleted,
		pipelineservice.JobStatusFailed,
	} {
		t.Run(status.String(), func(t *testing.T) {
			storage := newJobStateStorage(status)

			err := NewMarkJobRunning(storage).Execute(context.Background(), MarkJobRunningParams{ID: storage.job.ID})
			require.NoError(t, err)

			assert.Equal(t, status, storage.job.Status)
			assert.Empty(t, storage.events)
		})
	}
}

func TestMarkJobRunning_CancelledAfterReadIsUnchanged(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusCancelled)
	starting := *storage.job
	starting.Status = pipelineservice.JobStatusStarting
	storage.readJob = &starting

	err := NewMarkJobRunning(storage).Execute(context.Background(), MarkJobRunningParams{ID: storage.job.ID})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusCancelled, storage.job.Status)
	assert.Empty(t, storage.events)
}

func TestUpdateJobStatus_FailsRunningJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)
	storage.job.MaxAttempts = 1

	err := NewUpdateJobStatus(storage).Execute(context.Background(), UpdateJobStatusParams{
		ID:      storage.job.ID,
		SyncErr: errors.New("connector exited"),
	})
	require.NoError(t, err)

	assert.Equal(t, pipelineservice.JobStatusFailed, storage.job.Status)
	require.Len(t, storage.events, 1)
	assert.Equal(t, pipelineservice.JobStatusRunning, jobUpdatedData(t, storage.events[0]).OldData.Status)
}

func TestFailStaleJobs_FailsStaleRunningJob(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)
	storage.job.MaxAttempts = 1
	stale := time.Now().Add(-time.Hour)
	storage.job.HeartbeatAt = &stale

	uc := NewFailStaleJobs(nil, storage, NewUpdateJobStatus(storage), pipelineservice.Config{HeartbeatTimeout: time.Minute})

	failed, err := uc.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, failed)
	assert.Equal(t, pipelineservice.JobStatusFailed, storage.job.Status)
	require.NotNil(t, storage.job.Error)
	assert.Contains(t, *storage.job.Error, "heartbeat timeout")
}
