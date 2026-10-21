package pipelinecatalog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestDiscoverCatalog(t *testing.T) {
	discovered := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{Name: "users"}}}

	t.Run("first discovery is stored as version 1", func(t *testing.T) {
		store := newFakeStore()
		sourceID := uuid.New()

		uc := NewDiscoverCatalog(store, &fakeDiscoverer{catalog: discovered}, NewRecheckStreamMappers(store), testLogger())
		got, err := uc.Execute(context.Background(), DiscoverCatalogParams{SourceID: sourceID, Image: "img"})
		require.NoError(t, err)
		assert.Equal(t, discovered, got)

		require.Len(t, store.discoveries.records, 1)
		rec := store.discoveries.records[0]
		assert.Equal(t, sourceID, rec.SourceID)
		assert.Equal(t, 1, rec.Version)
		assert.JSONEq(t, `{"streams":[{"name":"users","json_schema":null}]}`, rec.CatalogJSON)
	})

	t.Run("subsequent discovery increments latest version", func(t *testing.T) {
		store := newFakeStore()
		sourceID := uuid.New()
		seedDiscovery(t, store, sourceID, 4, protocol.AirbyteCatalog{})
		seedDiscovery(t, store, uuid.New(), 9, protocol.AirbyteCatalog{})

		uc := NewDiscoverCatalog(store, &fakeDiscoverer{catalog: discovered}, NewRecheckStreamMappers(store), testLogger())
		_, err := uc.Execute(context.Background(), DiscoverCatalogParams{SourceID: sourceID})
		require.NoError(t, err)

		require.Len(t, store.discoveries.records, 3)
		assert.Equal(t, 5, store.discoveries.records[2].Version)
	})

	t.Run("connector failure stores nothing", func(t *testing.T) {
		store := newFakeStore()

		uc := NewDiscoverCatalog(store, &fakeDiscoverer{err: errStorage}, NewRecheckStreamMappers(store), testLogger())
		_, err := uc.Execute(context.Background(), DiscoverCatalogParams{SourceID: uuid.New()})
		require.ErrorIs(t, err, errStorage)
		assert.Empty(t, store.discoveries.records)
	})

	t.Run("latest lookup failure is returned", func(t *testing.T) {
		store := newFakeStore()
		store.discoveries.firstErr = errStorage

		uc := NewDiscoverCatalog(store, &fakeDiscoverer{catalog: discovered}, NewRecheckStreamMappers(store), testLogger())
		_, err := uc.Execute(context.Background(), DiscoverCatalogParams{SourceID: uuid.New()})
		require.ErrorIs(t, err, errStorage)
	})

	t.Run("store failure is returned", func(t *testing.T) {
		store := newFakeStore()
		store.discoveries.createErr = errStorage

		uc := NewDiscoverCatalog(store, &fakeDiscoverer{catalog: discovered}, NewRecheckStreamMappers(store), testLogger())
		_, err := uc.Execute(context.Background(), DiscoverCatalogParams{SourceID: uuid.New()})
		require.ErrorIs(t, err, errStorage)
	})

	t.Run("a discovery that breaks a connection's mappers pauses it", func(t *testing.T) {
		store, conn := recheckFixture(t, pipelineservice.ConnectionStatusActive)

		uc := NewDiscoverCatalog(store, &fakeDiscoverer{
			catalog: usersCatalog(`{"type":"object","properties":{"id":{"type":"integer"}}}`),
		}, NewRecheckStreamMappers(store), testLogger())
		_, err := uc.Execute(context.Background(), DiscoverCatalogParams{SourceID: conn.SourceID})
		require.NoError(t, err)

		got := store.connections.records[conn.ID]
		assert.Equal(t, pipelineservice.ConnectionStatusPaused, got.Status)
		require.NotNil(t, got.StatusReason)
		assert.Contains(t, *got.StatusReason, `stream "public.users": mapper step 1`)
	})
}

func TestDetectSchemaChanges(t *testing.T) {
	connectionID := uuid.New()
	sourceID := uuid.New()
	params := DetectSchemaChangesParams{ConnectionID: connectionID, SourceID: sourceID}

	latest := protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
		stream("", "users", `{"properties":{"id":{"type":"integer"},"email":{"type":"string"}}}`),
	}}

	t.Run("reports differences against configured streams", func(t *testing.T) {
		store := newFakeStore()
		seedDiscovery(t, store, sourceID, 1, latest)
		seedConfigured(t, store, connectionID, []protocol.ConfiguredAirbyteStream{
			{Stream: stream("", "users", `{"properties":{"id":{"type":"integer"}}}`)},
		})

		changes, err := NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.NoError(t, err)
		assert.Equal(t, []SchemaChange{{Type: ColumnAdded, StreamName: "users", ColumnName: "email"}}, changes)
	})

	t.Run("no changes returns nil", func(t *testing.T) {
		store := newFakeStore()
		seedDiscovery(t, store, sourceID, 1, latest)
		seedConfigured(t, store, connectionID, []protocol.ConfiguredAirbyteStream{{Stream: latest.Streams[0]}})

		changes, err := NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, changes)
	})

	t.Run("unconfigured connection has no changes", func(t *testing.T) {
		store := newFakeStore()
		seedDiscovery(t, store, sourceID, 1, latest)

		changes, err := NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.NoError(t, err)
		assert.Nil(t, changes)
	})

	t.Run("errors", func(t *testing.T) {
		store := newFakeStore()
		_, err := NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.ErrorIs(t, err, pipelineservice.ErrCatalogDiscoveryNotFound)

		seedDiscovery(t, store, sourceID, 1, latest)
		store.configured.firstErr = errStorage
		_, err = NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.ErrorIs(t, err, errStorage)

		store.configured.firstErr = nil
		store.configured.records[connectionID] = &pipelineservice.ConfiguredCatalog{ConnectionID: connectionID, StreamsJSON: "{"}
		_, err = NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.ErrorContains(t, err, "unmarshaling configured streams")

		store.discoveries.records[0].CatalogJSON = "{"
		_, err = NewDetectSchemaChanges(store, nil).Execute(context.Background(), params)
		require.ErrorContains(t, err, "unmarshaling latest catalog")
	})
}

func TestGetConfiguredCatalog(t *testing.T) {
	newStore := func(t *testing.T) (*fakeStore, *pipelineservice.Connection) {
		t.Helper()

		store := newFakeStore()
		conn := &pipelineservice.Connection{ID: uuid.New(), SourceID: uuid.New()}
		store.connections.records[conn.ID] = conn
		seedConfigured(t, store, conn.ID, []protocol.ConfiguredAirbyteStream{
			configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
			configured("public", "gone", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
		})

		return store, conn
	}

	t.Run("enriches streams from latest discovery", func(t *testing.T) {
		store, conn := newStore(t)
		seedDiscovery(t, store, conn.SourceID, 1, protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{
			Namespace:               "public",
			Name:                    "users",
			JSONSchema:              json.RawMessage(nestedSchema),
			SupportedSyncModes:      []protocol.SyncMode{protocol.SyncModeFullRefresh},
			SourceDefinedCursor:     true,
			DefaultCursorField:      []string{"updated_at"},
			SourceDefinedPrimaryKey: [][]string{{"id"}},
		}}})

		got, err := NewGetConfiguredCatalog(store).Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: conn.ID})
		require.NoError(t, err)
		require.Len(t, got.Streams, 2)

		users := got.Streams[0].Stream
		assert.JSONEq(t, nestedSchema, string(users.JSONSchema))
		assert.True(t, users.SourceDefinedCursor)
		assert.Equal(t, []string{"updated_at"}, users.DefaultCursorField)
		assert.Equal(t, [][]string{{"id"}}, users.SourceDefinedPrimaryKey)

		assert.JSONEq(t, "null", string(got.Streams[1].Stream.JSONSchema), "streams absent from discovery are left as stored")
	})

	t.Run("missing or corrupt discovery returns stored streams", func(t *testing.T) {
		store, conn := newStore(t)

		got, err := NewGetConfiguredCatalog(store).Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: conn.ID})
		require.NoError(t, err)
		assert.Len(t, got.Streams, 2)

		store.discoveries.records = []*pipelineservice.CatalogDiscovery{{SourceID: conn.SourceID, CatalogJSON: "{"}}

		got, err = NewGetConfiguredCatalog(store).Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: conn.ID})
		require.NoError(t, err)
		assert.Len(t, got.Streams, 2)
	})

	t.Run("errors", func(t *testing.T) {
		store, conn := newStore(t)
		uc := NewGetConfiguredCatalog(store)

		_, err := uc.Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: uuid.New()})
		require.ErrorIs(t, err, pipelineservice.ErrConfiguredCatalogNotFound)

		store.configured.records[conn.ID].StreamsJSON = "{"
		_, err = uc.Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: conn.ID})
		require.ErrorContains(t, err, "unmarshaling configured streams")

		store.configured.records[conn.ID].StreamsJSON = "[]"
		delete(store.connections.records, conn.ID)
		_, err = uc.Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: conn.ID})
		require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
	})
}

func TestGetDiscoveredCatalogForConnection(t *testing.T) {
	workspaceID := uuid.New()
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: workspaceID, SourceID: uuid.New()}
	params := GetDiscoveredCatalogForConnectionParams{ConnectionID: conn.ID, WorkspaceID: workspaceID}

	newUC := func(store *fakeStore) *GetDiscoveredCatalogForConnection {
		return NewGetDiscoveredCatalogForConnection(store)
	}

	t.Run("returns latest cached catalog", func(t *testing.T) {
		store := newFakeStore()
		store.connections.records[conn.ID] = conn
		seedDiscovery(t, store, conn.SourceID, 1, protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{Name: "old"}}})
		seedDiscovery(t, store, conn.SourceID, 2, protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{Name: "new"}}})

		got, err := newUC(store).Execute(context.Background(), params)
		require.NoError(t, err)
		require.Len(t, got.Streams, 1)
		assert.Equal(t, "new", got.Streams[0].Name)
	})

	t.Run("connection in another workspace is not found", func(t *testing.T) {
		store := newFakeStore()
		store.connections.records[conn.ID] = conn

		_, err := newUC(store).Execute(context.Background(), GetDiscoveredCatalogForConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  uuid.New(),
		})
		require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
	})

	t.Run("missing or corrupt cache", func(t *testing.T) {
		store := newFakeStore()
		store.connections.records[conn.ID] = conn

		_, err := newUC(store).Execute(context.Background(), params)
		require.ErrorIs(t, err, pipelineservice.ErrCatalogDiscoveryNotFound)

		store.discoveries.records = []*pipelineservice.CatalogDiscovery{{SourceID: conn.SourceID, CatalogJSON: "{"}}
		_, err = newUC(store).Execute(context.Background(), params)
		require.ErrorContains(t, err, "parsing cached catalog")
	})
}

func TestGetSourceCatalog(t *testing.T) {
	workspaceID := uuid.New()
	source := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: workspaceID}
	params := GetSourceCatalogParams{SourceID: source.ID, WorkspaceID: workspaceID}

	newStore := func() *fakeStore {
		store := newFakeStore()
		store.sources.records[source.ID] = source

		return store
	}

	t.Run("returns latest version with metadata", func(t *testing.T) {
		store := newStore()
		seedDiscovery(t, store, source.ID, 1, protocol.AirbyteCatalog{})
		seedDiscovery(t, store, source.ID, 3, protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{Name: "users"}}})

		discoveredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		store.discoveries.records[1].DiscoveredAt = discoveredAt

		got, err := NewGetSourceCatalog(store).Execute(context.Background(), params)
		require.NoError(t, err)
		assert.Equal(t, 3, got.Version)
		assert.Equal(t, discoveredAt, got.DiscoveredAt)
		require.Len(t, got.Catalog.Streams, 1)
		assert.Equal(t, "users", got.Catalog.Streams[0].Name)
	})

	t.Run("source in another workspace is not found", func(t *testing.T) {
		_, err := NewGetSourceCatalog(newStore()).Execute(context.Background(), GetSourceCatalogParams{
			SourceID:    source.ID,
			WorkspaceID: uuid.New(),
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})

	t.Run("no discovery yields not-found", func(t *testing.T) {
		_, err := NewGetSourceCatalog(newStore()).Execute(context.Background(), params)

		var nfe pipelineservice.NotFoundError
		require.ErrorAs(t, err, &nfe)
	})

	t.Run("lookup and parse errors", func(t *testing.T) {
		store := newStore()
		store.discoveries.firstErr = errStorage
		_, err := NewGetSourceCatalog(store).Execute(context.Background(), params)
		require.ErrorIs(t, err, errStorage)

		store.discoveries.firstErr = nil
		store.discoveries.records = []*pipelineservice.CatalogDiscovery{{SourceID: source.ID, CatalogJSON: "{"}}
		_, err = NewGetSourceCatalog(store).Execute(context.Background(), params)
		require.ErrorContains(t, err, "parsing cached catalog")
	})
}
