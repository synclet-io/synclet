package pipelinecatalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func recheckFixture(t *testing.T, status pipelineservice.ConnectionStatus) (*fakeStore, *pipelineservice.Connection) {
	t.Helper()

	store := newFakeStore()
	conn := &pipelineservice.Connection{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		SourceID:    uuid.New(),
		Name:        "pg-to-bq",
		Status:      status,
	}
	store.connections.records[conn.ID] = conn

	users := configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite)
	users.Stream.JSONSchema = json.RawMessage(nestedSchema)
	users.Mappers = json.RawMessage(`[{"rename":{"id":"user_id"}},{"set":{"label":"r.name + '!'"}}]`)
	seedConfigured(t, store, conn.ID, []protocol.ConfiguredAirbyteStream{users})

	return store, conn
}

func usersCatalog(schema string) *protocol.AirbyteCatalog {
	return &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{
		Namespace:  "public",
		Name:       "users",
		JSONSchema: json.RawMessage(schema),
	}}}
}

func TestRecheckStreamMappers(t *testing.T) {
	t.Run("a discovery dropping a field a set reads pauses the connection", func(t *testing.T) {
		store, conn := recheckFixture(t, pipelineservice.ConnectionStatusActive)

		err := NewRecheckStreamMappers(store).Execute(context.Background(), conn.SourceID,
			usersCatalog(`{"type":"object","properties":{"id":{"type":"integer"}}}`))
		require.NoError(t, err)

		got := store.connections.records[conn.ID]
		assert.Equal(t, pipelineservice.ConnectionStatusPaused, got.Status)
		require.NotNil(t, got.StatusReason)
		assert.Contains(t, *got.StatusReason, `stream "public.users": mapper step 1`)
		assert.Contains(t, *got.StatusReason, "name")
		assert.Nil(t, got.NextScheduledAt)
	})

	t.Run("a discovery adding a column leaves the connection active", func(t *testing.T) {
		store, conn := recheckFixture(t, pipelineservice.ConnectionStatusActive)

		err := NewRecheckStreamMappers(store).Execute(context.Background(), conn.SourceID,
			usersCatalog(`{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}}}`))
		require.NoError(t, err)

		got := store.connections.records[conn.ID]
		assert.Equal(t, pipelineservice.ConnectionStatusActive, got.Status)
		assert.Nil(t, got.StatusReason)
		assert.Zero(t, store.connections.updated)
	})

	t.Run("a paused connection is untouched", func(t *testing.T) {
		store, conn := recheckFixture(t, pipelineservice.ConnectionStatusPaused)

		err := NewRecheckStreamMappers(store).Execute(context.Background(), conn.SourceID,
			usersCatalog(`{"type":"object","properties":{"id":{"type":"integer"}}}`))
		require.NoError(t, err)

		got := store.connections.records[conn.ID]
		assert.Nil(t, got.StatusReason)
		assert.Zero(t, store.connections.updated)
	})

	t.Run("a stream gone from the discovery is skipped", func(t *testing.T) {
		store, conn := recheckFixture(t, pipelineservice.ConnectionStatusActive)

		err := NewRecheckStreamMappers(store).Execute(context.Background(), conn.SourceID, &protocol.AirbyteCatalog{})
		require.NoError(t, err)

		assert.Equal(t, pipelineservice.ConnectionStatusActive, store.connections.records[conn.ID].Status)
	})

	t.Run("a connection without a configured catalog is skipped", func(t *testing.T) {
		store, conn := recheckFixture(t, pipelineservice.ConnectionStatusActive)
		delete(store.configured.records, conn.ID)

		err := NewRecheckStreamMappers(store).Execute(context.Background(), conn.SourceID,
			usersCatalog(`{"type":"object","properties":{"id":{"type":"integer"}}}`))
		require.NoError(t, err)

		assert.Equal(t, pipelineservice.ConnectionStatusActive, store.connections.records[conn.ID].Status)
	})
}
