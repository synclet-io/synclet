package pipelineconnections_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

func TestConnectionExecutorGroup(t *testing.T) {
	ctx := context.Background()

	t.Run("create rejects a source and destination of different groups", func(t *testing.T) {
		ws, sourceID, destID := uuid.New(), uuid.New(), uuid.New()
		store := newCreateConnStore(ws, sourceID, destID)
		store.sources.records[sourceID].ExecutorGroup = "eu"
		store.destinations.records[destID].ExecutorGroup = "us"

		_, err := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{}).
			Execute(ctx, pipelineconnections.CreateConnectionParams{WorkspaceID: ws, Name: "c", SourceID: sourceID, DestinationID: destID})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
		assert.Equal(t, `source group "eu" and destination group "us" differ`, validationErr.Message)
		assert.Empty(t, store.connections.records)
	})

	t.Run("create accepts a source and destination of one group", func(t *testing.T) {
		ws, sourceID, destID := uuid.New(), uuid.New(), uuid.New()
		store := newCreateConnStore(ws, sourceID, destID)
		store.sources.records[sourceID].ExecutorGroup = "us"
		store.destinations.records[destID].ExecutorGroup = "us"

		_, err := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{}).
			Execute(ctx, pipelineconnections.CreateConnectionParams{WorkspaceID: ws, Name: "c", SourceID: sourceID, DestinationID: destID})
		require.NoError(t, err)
	})

	t.Run("update rejects a connection whose source and destination groups differ", func(t *testing.T) {
		ws, sourceID, destID := uuid.New(), uuid.New(), uuid.New()
		store := newCreateConnStore(ws, sourceID, destID)
		store.sources.records[sourceID].ExecutorGroup = "eu"
		store.destinations.records[destID].ExecutorGroup = "us"
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Name: "old", SourceID: sourceID, DestinationID: destID}
		seedConnection(t, store, conn)

		name := "new"
		_, err := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{}).
			Execute(ctx, pipelineconnections.UpdateConnectionParams{ID: conn.ID, WorkspaceID: ws, Name: &name})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
		assert.Equal(t, "old", store.connections.records[conn.ID].Name)
	})
}
