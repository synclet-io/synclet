package pipelinesync

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// mockSchedulerStorage implements the subset of pipelineservice.Storage that SyncScheduler needs.
type mockSchedulerStorage struct {
	lockAcquired       bool
	lockErr            error
	activeCount        int
	activeCountErr     error
	dueConnections     []pipelineservice.DueConnection
	dueErr             error
	createdJobs        []*pipelineservice.Job
	createErr          error
	updatedConnections []*pipelineservice.Connection
	txExecuted         bool
}

// mockSchedulerConnectionsStorage implements pipelineservice.ConnectionsStorage for test.
type mockSchedulerConnectionsStorage struct {
	pipelineservice.ConnectionsStorage
	parent *mockSchedulerStorage
}

func (m *mockSchedulerConnectionsStorage) FindDueConnections(ctx context.Context, limit int) ([]pipelineservice.DueConnection, error) {
	if m.parent.dueErr != nil {
		return nil, m.parent.dueErr
	}

	if limit >= len(m.parent.dueConnections) {
		return m.parent.dueConnections, nil
	}

	return m.parent.dueConnections[:limit], nil
}

func (m *mockSchedulerConnectionsStorage) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	// Extract the UUID from the Equals filter.
	if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok {
		return &pipelineservice.Connection{ID: eq.Value}, nil
	}

	return &pipelineservice.Connection{}, nil
}

func (m *mockSchedulerConnectionsStorage) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	// Return a Connection for each due connection ID (simulates batch load).
	result := make([]*pipelineservice.Connection, 0, len(m.parent.dueConnections))
	for _, dc := range m.parent.dueConnections {
		result = append(result, &pipelineservice.Connection{
			ID:       dc.ConnectionID,
			Status:   pipelineservice.ConnectionStatusActive,
			Schedule: &dc.Schedule,
		})
	}

	return result, nil
}

func (m *mockSchedulerConnectionsStorage) Update(_ context.Context, conn *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	m.parent.updatedConnections = append(m.parent.updatedConnections, conn)

	return conn, nil
}

// mockSchedulerStorageWrapper wraps mockSchedulerStorage to implement pipelineservice.Storage.
// We only implement the methods SyncScheduler uses.
type mockSchedulerStorageWrapper struct {
	pipelineservice.Storage
	mock            *mockSchedulerStorage
	connectionsMock *mockSchedulerConnectionsStorage
}

func (w *mockSchedulerStorageWrapper) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	w.mock.txExecuted = true

	return cb(ctx, w)
}

func (w *mockSchedulerStorageWrapper) WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error {
	if !w.mock.lockAcquired {
		return fmt.Errorf("lock not acquired")
	}

	return w.mock.lockErr
}

func (w *mockSchedulerStorageWrapper) Connections() pipelineservice.ConnectionsStorage {
	return w.connectionsMock
}

func (w *mockSchedulerStorageWrapper) Jobs() pipelineservice.JobsStorage {
	return &mockSchedulerJobsStorageWrapper{mock: w.mock}
}

// mockSchedulerJobsStorageWrapper implements pipelineservice.JobsStorage for test.
type mockSchedulerJobsStorageWrapper struct {
	pipelineservice.JobsStorage
	mock *mockSchedulerStorage
}

func (m *mockSchedulerJobsStorageWrapper) CountActiveJobs(ctx context.Context) (int, error) {
	return m.mock.activeCount, m.mock.activeCountErr
}

func (m *mockSchedulerJobsStorageWrapper) Create(ctx context.Context, job *pipelineservice.Job) (*pipelineservice.Job, error) {
	if m.mock.createErr != nil {
		return nil, m.mock.createErr
	}

	m.mock.createdJobs = append(m.mock.createdJobs, job)

	return job, nil
}

func TestSyncScheduler_Execute(t *testing.T) {
	t.Run("acquires lock and creates scheduled jobs for due connections", func(t *testing.T) {
		mock := &mockSchedulerStorage{
			lockAcquired: true,
			activeCount:  2,
			dueConnections: []pipelineservice.DueConnection{
				{ConnectionID: uuid.New(), SourceID: uuid.New(), DestinationID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3},
				{ConnectionID: uuid.New(), SourceID: uuid.New(), DestinationID: uuid.New(), Schedule: "0 * * * *", MaxAttempts: 3},
			},
		}
		storage := &mockSchedulerStorageWrapper{mock: mock, connectionsMock: &mockSchedulerConnectionsStorage{parent: mock}}

		scheduler := NewSyncScheduler(storage, 10, nil)
		err := scheduler.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, mock.txExecuted, "should execute in transaction")
		assert.Len(t, mock.createdJobs, 2, "should create 2 jobs")

		for _, job := range mock.createdJobs {
			assert.Equal(t, pipelineservice.JobStatusScheduled, job.Status)
			assert.Equal(t, pipelineservice.JobTypeSync, job.JobType)
		}

		assert.Len(t, mock.updatedConnections, 2, "should update next_scheduled_at for 2 connections")

		for _, conn := range mock.updatedConnections {
			assert.NotNil(t, conn.NextScheduledAt, "next_scheduled_at should be set")
		}
	})

	t.Run("returns error when advisory lock not acquired", func(t *testing.T) {
		mock := &mockSchedulerStorage{
			lockAcquired: false,
		}
		storage := &mockSchedulerStorageWrapper{mock: mock, connectionsMock: &mockSchedulerConnectionsStorage{parent: mock}}

		scheduler := NewSyncScheduler(storage, 10, nil)
		err := scheduler.Execute(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "acquiring scheduler lock")
		assert.True(t, mock.txExecuted, "should still enter transaction")
		assert.Empty(t, mock.createdJobs, "should not create any jobs")
	})

	t.Run("creates zero jobs when active count at limit", func(t *testing.T) {
		mock := &mockSchedulerStorage{
			lockAcquired: true,
			activeCount:  10,
			dueConnections: []pipelineservice.DueConnection{
				{ConnectionID: uuid.New()},
			},
		}
		storage := &mockSchedulerStorageWrapper{mock: mock, connectionsMock: &mockSchedulerConnectionsStorage{parent: mock}}

		scheduler := NewSyncScheduler(storage, 10, nil)
		err := scheduler.Execute(context.Background())

		require.NoError(t, err)
		assert.Empty(t, mock.createdJobs, "should not create any jobs when at limit")
	})

	t.Run("respects remaining capacity", func(t *testing.T) {
		mock := &mockSchedulerStorage{
			lockAcquired: true,
			activeCount:  8,
			dueConnections: []pipelineservice.DueConnection{
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3},
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3},
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3},
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3},
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3},
			},
		}
		storage := &mockSchedulerStorageWrapper{mock: mock, connectionsMock: &mockSchedulerConnectionsStorage{parent: mock}}

		scheduler := NewSyncScheduler(storage, 10, nil)
		err := scheduler.Execute(context.Background())

		require.NoError(t, err)
		// limit=10, active=8, remaining=2, but FindDueConnections returns min(2, 5)=2
		assert.Len(t, mock.createdJobs, 2, "should only create jobs up to remaining capacity")
	})
}
