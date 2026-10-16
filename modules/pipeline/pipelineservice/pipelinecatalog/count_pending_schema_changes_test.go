package pipelinecatalog

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestCountPendingSchemaChanges(t *testing.T) {
	latest := protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
		stream("", "users", `{"properties":{"id":{"type":"integer"},"email":{"type":"string"},"name":{"type":"string"}}}`),
	}}
	drifted := []protocol.ConfiguredAirbyteStream{{Stream: stream("", "users", `{"properties":{"id":{"type":"integer"}}}`)}}
	current := []protocol.ConfiguredAirbyteStream{{Stream: latest.Streams[0]}}

	t.Run("counts changes per connection and reads each source once", func(t *testing.T) {
		store := newFakeStore()
		sourceA, sourceB := uuid.New(), uuid.New()
		driftedA, cleanA, unconfigured, undiscovered := uuid.New(), uuid.New(), uuid.New(), uuid.New()

		seedDiscovery(t, store, sourceA, 1, protocol.AirbyteCatalog{})
		seedDiscovery(t, store, sourceA, 2, latest)
		seedConfigured(t, store, driftedA, drifted)
		seedConfigured(t, store, cleanA, current)
		seedConfigured(t, store, undiscovered, drifted)

		counts, err := NewCountPendingSchemaChanges(store).Execute(context.Background(), []PendingSchemaChangesTarget{
			{ConnectionID: driftedA, SourceID: sourceA},
			{ConnectionID: cleanA, SourceID: sourceA},
			{ConnectionID: unconfigured, SourceID: sourceA},
			{ConnectionID: undiscovered, SourceID: sourceB},
		})
		require.NoError(t, err)
		assert.Equal(t, map[uuid.UUID]int{driftedA: 2}, counts)
		assert.Equal(t, 2, store.discoveries.firstCalls)
	})

	t.Run("matches DetectSchemaChanges", func(t *testing.T) {
		store := newFakeStore()
		sourceID, connID := uuid.New(), uuid.New()
		seedDiscovery(t, store, sourceID, 1, latest)
		seedConfigured(t, store, connID, drifted)

		changes, err := NewDetectSchemaChanges(store, nil).Execute(context.Background(), DetectSchemaChangesParams{ConnectionID: connID, SourceID: sourceID})
		require.NoError(t, err)

		counts, err := NewCountPendingSchemaChanges(store).Execute(context.Background(), []PendingSchemaChangesTarget{{ConnectionID: connID, SourceID: sourceID}})
		require.NoError(t, err)
		assert.Equal(t, len(changes), counts[connID])
	})

	t.Run("no targets skips storage", func(t *testing.T) {
		store := newFakeStore()
		store.configured.firstErr = errStorage

		counts, err := NewCountPendingSchemaChanges(store).Execute(context.Background(), nil)
		require.NoError(t, err)
		assert.Empty(t, counts)
	})

	t.Run("errors", func(t *testing.T) {
		store := newFakeStore()
		sourceID, connID := uuid.New(), uuid.New()
		targets := []PendingSchemaChangesTarget{{ConnectionID: connID, SourceID: sourceID}}

		store.configured.firstErr = errStorage
		_, err := NewCountPendingSchemaChanges(store).Execute(context.Background(), targets)
		require.ErrorIs(t, err, errStorage)

		store.configured.firstErr = nil
		seedConfigured(t, store, connID, drifted)
		store.discoveries.firstErr = errStorage
		_, err = NewCountPendingSchemaChanges(store).Execute(context.Background(), targets)
		require.ErrorIs(t, err, errStorage)

		store.discoveries.firstErr = nil
		seedDiscovery(t, store, sourceID, 1, latest)
		store.discoveries.records[0].CatalogJSON = "{"
		_, err = NewCountPendingSchemaChanges(store).Execute(context.Background(), targets)
		require.ErrorContains(t, err, "unmarshaling latest catalog")
	})
}
