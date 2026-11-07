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

func newGroupQueueFixture(t *testing.T, group string) (*pipelinefake.Storage, *pipelineservice.Connection) {
	t.Helper()

	store := pipelinefake.New()
	workspaceID := uuid.New()
	source := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: workspaceID, ExecutorGroup: group}
	store.SourcesTable.Insert(source)

	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: workspaceID, SourceID: source.ID, Status: pipelineservice.ConnectionStatusActive, MaxAttempts: 2}
	store.ConnectionsTable.Insert(conn)
	store.ConfigureStreams(conn.ID, "users")

	return store, conn
}

func TestQueueJob_ExecutorGroup(t *testing.T) {
	cfg := pipelineservice.Config{ExecutorSilenceTimeout: time.Minute}

	t.Run("the job records its source's group and clears a no-live reason", func(t *testing.T) {
		store, conn := newGroupQueueFixture(t, "us")
		store.AddLiveExecutor("us")

		reason := `no live executor in group "us"`
		store.ConnectionsTable.Get(conn.ID).StatusReason = &reason

		job, err := NewQueueJob(store, cfg).Execute(context.Background(), QueueJobParams{ConnectionID: conn.ID, JobType: pipelineservice.JobTypeSync})
		require.NoError(t, err)
		assert.Equal(t, "us", job.ExecutorGroup)
		assert.Equal(t, "us", store.JobsTable.Get(job.ID).ExecutorGroup)
		assert.Nil(t, store.ConnectionsTable.Get(conn.ID).StatusReason)
	})

	t.Run("manual sync with no live executor in the group", func(t *testing.T) {
		store, conn := newGroupQueueFixture(t, "us")
		store.AddLiveExecutor("eu")

		_, err := NewQueueJob(store, cfg).Execute(context.Background(), QueueJobParams{ConnectionID: conn.ID, JobType: pipelineservice.JobTypeSync})

		var noLive *pipelineservice.NoLiveExecutorError
		require.ErrorAs(t, err, &noLive)
		require.EqualError(t, noLive, `no live executor in group "us"`)
		assert.Empty(t, store.JobsTable.All())
	})
}
