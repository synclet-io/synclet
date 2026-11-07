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

func TestSyncScheduler_ExecutorGroup(t *testing.T) {
	cfg := pipelineservice.Config{MaxConcurrentJobs: 10, ExecutorSilenceTimeout: time.Minute}

	t.Run("a due connection's job carries its source's group", func(t *testing.T) {
		mock := &mockSchedulerStorage{
			lockAcquired: true,
			dueConnections: []pipelineservice.DueConnection{
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3, ExecutorGroup: "us"},
			},
		}
		storage := &mockSchedulerStorageWrapper{mock: mock, connectionsMock: &mockSchedulerConnectionsStorage{parent: mock}}

		require.NoError(t, NewSyncScheduler(storage, cfg, nil).Execute(context.Background()))
		require.Len(t, mock.createdJobs, 1)
		assert.Equal(t, "us", mock.createdJobs[0].ExecutorGroup)
	})

	t.Run("no live executor in the group: no job, reason set, next run advanced", func(t *testing.T) {
		connID := uuid.New()
		mock := &mockSchedulerStorage{
			lockAcquired: true,
			deadGroups:   map[string]bool{"us": true},
			dueConnections: []pipelineservice.DueConnection{
				{ConnectionID: connID, Schedule: "*/5 * * * *", MaxAttempts: 3, ExecutorGroup: "us"},
				{ConnectionID: uuid.New(), Schedule: "*/5 * * * *", MaxAttempts: 3, ExecutorGroup: "eu"},
			},
		}
		storage := &mockSchedulerStorageWrapper{mock: mock, connectionsMock: &mockSchedulerConnectionsStorage{parent: mock}}

		require.NoError(t, NewSyncScheduler(storage, cfg, nil).Execute(context.Background()))

		require.Len(t, mock.createdJobs, 1, "only the eu connection gets a job")
		assert.Equal(t, "eu", mock.createdJobs[0].ExecutorGroup)

		var skipped *pipelineservice.Connection

		for _, conn := range mock.updatedConnections {
			if conn.ID == connID {
				skipped = conn
			}
		}

		require.NotNil(t, skipped, "the skipped connection is updated")
		require.NotNil(t, skipped.StatusReason)
		assert.Equal(t, `no live executor in group "us"`, *skipped.StatusReason)
		assert.Equal(t, pipelineservice.ConnectionStatusActive, skipped.Status)
		require.NotNil(t, skipped.NextScheduledAt)
		assert.True(t, skipped.NextScheduledAt.After(time.Now()))
	})
}
