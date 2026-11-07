package pipelinetasks_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

func TestTaskExecutorGroup(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("check of a source runs in the source's group", func(t *testing.T) {
		store := newTaskStore()
		store.executors.records = append(store.executors.records, liveExecutor("us"))
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New(), ExecutorGroup: "us"}
		store.sources.records[src.ID] = src

		got, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID: ws,
			SourceID:    &src.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, new("us"), store.tasks.get(got.TaskID).ExecutorGroup)
	})

	t.Run("check of a destination runs in the destination's group", func(t *testing.T) {
		store := newTaskStore()
		store.executors.records = append(store.executors.records, liveExecutor("eu"))
		dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New(), ExecutorGroup: "eu"}
		store.destinations.records[dst.ID] = dst

		got, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:   ws,
			DestinationID: &dst.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, new("eu"), store.tasks.get(got.TaskID).ExecutorGroup)
	})

	t.Run("unsaved check runs in the group picked in the form", func(t *testing.T) {
		store := newTaskStore()
		store.executors.records = append(store.executors.records, liveExecutor("us"))
		mc := seedConnector(store, ws)

		got, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:        ws,
			ManagedConnectorID: &mc.ID,
			Config:             json.RawMessage(`{"host":"db"}`),
			ExecutorGroup:      "  us ",
		})
		require.NoError(t, err)
		assert.Equal(t, new("us"), store.tasks.get(got.TaskID).ExecutorGroup)
	})

	t.Run("unsaved check without a group runs in default", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)

		got, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:        ws,
			ManagedConnectorID: &mc.ID,
			Config:             json.RawMessage(`{"host":"db"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, new(pipelineservice.DefaultExecutorGroup), store.tasks.get(got.TaskID).ExecutorGroup)
	})

	t.Run("discover runs in the source's group", func(t *testing.T) {
		store := newTaskStore()
		store.executors.records = append(store.executors.records, liveExecutor("us"))
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New(), ExecutorGroup: "us"}
		store.sources.records[src.ID] = src

		got, err := pipelinetasks.NewCreateDiscoverTask(store, &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.NoError(t, err)
		assert.Equal(t, new("us"), store.tasks.get(got.TaskID).ExecutorGroup)
	})

	t.Run("spec runs on any executor", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)

		got, err := pipelinetasks.NewCreateSpecTask(store).Execute(ctx, pipelinetasks.CreateSpecTaskParams{WorkspaceID: ws, ManagedConnectorID: mc.ID})
		require.NoError(t, err)
		assert.Nil(t, store.tasks.get(got.TaskID).ExecutorGroup)
	})
}

func TestTaskNoLiveExecutor(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("check of a source in a group without live executors", func(t *testing.T) {
		store := newTaskStore()
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New(), ExecutorGroup: "us"}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID: ws,
			SourceID:    &src.ID,
		})

		var noLive *pipelineservice.NoLiveExecutorError
		require.ErrorAs(t, err, &noLive)
		require.EqualError(t, noLive, `no live executor in group "us"`)
		assert.Empty(t, store.tasks.records)
	})

	t.Run("unsaved check in a group whose executors went silent", func(t *testing.T) {
		store := newTaskStore()
		silent := liveExecutor("us")
		silent.LastSeenAt = silent.LastSeenAt.Add(-2 * taskCfg.ExecutorSilenceTimeout)
		store.executors.records = append(store.executors.records, silent)
		mc := seedConnector(store, ws)

		_, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:        ws,
			ManagedConnectorID: &mc.ID,
			Config:             json.RawMessage(`{"host":"db"}`),
			ExecutorGroup:      "us",
		})

		var noLive *pipelineservice.NoLiveExecutorError
		require.ErrorAs(t, err, &noLive)
		assert.Empty(t, store.tasks.records)
	})

	t.Run("discover of a source in a group without live executors", func(t *testing.T) {
		store := newTaskStore()
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New(), ExecutorGroup: "us"}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateDiscoverTask(store, &recordingAudit{}, taskCfg).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})

		var noLive *pipelineservice.NoLiveExecutorError
		require.ErrorAs(t, err, &noLive)
		require.EqualError(t, noLive, `no live executor in group "us"`)
	})
}
