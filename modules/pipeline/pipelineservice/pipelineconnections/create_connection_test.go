package pipelineconnections_test

import (
	"context"
	"testing"

	"encoding/json"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// TestCreateConnection_DoesNotCarryStateFromAnotherConnection verifies that the
// CreateConnection use case never touches sync state — even when the wizard
// pre-populates fields from an existing connection (the frontend "Duplicate"
// flow). The cloned connection must start with zero stream state and zero
// stream generation rows.
func TestCreateConnection_DoesNotCarryStateFromAnotherConnection(t *testing.T) {
	workspaceID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	originalConnID := uuid.New()

	store := newCreateConnStore(workspaceID, sourceID, destID)

	// Seed the storage with an existing connection that has state — this is the
	// connection the user is "cloning from" on the UI side. The clone must NOT
	// inherit this state.
	store.connections.records[originalConnID] = &pipelineservice.Connection{
		ID:            originalConnID,
		WorkspaceID:   workspaceID,
		Name:          "original",
		SourceID:      sourceID,
		DestinationID: destID,
		Status:        pipelineservice.ConnectionStatusActive,
	}
	store.connStates.records[originalConnID] = &pipelineservice.ConnectionState{
		ConnectionID: originalConnID,
		StateType:    "stream",
		StateBlob:    `[{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":"100"}}}]`,
	}

	uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{})
	created, err := uc.Execute(context.Background(), pipelineconnections.CreateConnectionParams{
		WorkspaceID:   workspaceID,
		Name:          "original (copy)",
		SourceID:      sourceID,
		DestinationID: destID,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEqual(t, originalConnID, created.ID, "clone must have its own ID")
	assert.Equal(t, pipelineservice.ConnectionStatusActive, created.Status)

	// Original state is untouched.
	_, ok := store.connStates.records[originalConnID]
	assert.True(t, ok, "original connection state must remain")

	// Cloned connection has no state row.
	_, ok = store.connStates.records[created.ID]
	assert.False(t, ok, "cloned connection must not have a ConnectionState row")

	// Cloned connection has no stream generation rows.
	for _, gen := range store.streamGens.records {
		assert.NotEqual(t, created.ID, gen.ConnectionID, "cloned connection must not have StreamGeneration rows")
	}
}

func TestCreateConnection_InvalidCronIsValidationError(t *testing.T) {
	workspaceID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	store := newCreateConnStore(workspaceID, sourceID, destID)
	schedule := "every hour"

	uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{})
	_, err := uc.Execute(context.Background(), pipelineconnections.CreateConnectionParams{
		WorkspaceID:   workspaceID,
		Name:          "hourly",
		SourceID:      sourceID,
		DestinationID: destID,
		Schedule:      &schedule,
	})

	var validationErr *pipelineservice.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "schedule", validationErr.Field)
	assert.Empty(t, store.connections.records)
}

func TestCreateConnection_HonoursRequestedStatus(t *testing.T) {
	workspaceID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	store := newCreateConnStore(workspaceID, sourceID, destID)
	schedule := "0 * * * *"

	uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{})
	created, err := uc.Execute(context.Background(), pipelineconnections.CreateConnectionParams{
		WorkspaceID:   workspaceID,
		Name:          "paused",
		SourceID:      sourceID,
		DestinationID: destID,
		Schedule:      &schedule,
		Status:        pipelineservice.ConnectionStatusInactive,
	})
	require.NoError(t, err)
	assert.Equal(t, pipelineservice.ConnectionStatusInactive, created.Status)
	assert.Nil(t, created.NextScheduledAt)
}

// ---- minimal in-memory storage scaffolding for the CreateConnection use case ----

type createConnStore struct {
	pipelineservice.Storage
	sources      *createConnSourcesStorage
	destinations *createConnDestinationsStorage
	connections  *createConnConnectionsStorage
	connStates   *createConnConnectionStatesStorage
	streamGens   *createConnStreamGenerationsStorage
	fake         *pipelinefake.Storage
}

func (s *createConnStore) WorkflowSteps() pipelineservice.WorkflowStepsStorage {
	return s.fake.WorkflowSteps()
}

func (s *createConnStore) CatalogDiscoveries() pipelineservice.CatalogDiscoveriesStorage {
	return s.fake.CatalogDiscoveries()
}

func (s *createConnStore) ConfiguredCatalogs() pipelineservice.ConfiguredCatalogsStorage {
	return s.fake.ConfiguredCatalogs()
}

func (s *createConnStore) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

func newCreateConnStore(workspaceID, sourceID, destID uuid.UUID) *createConnStore {
	return &createConnStore{
		sources: &createConnSourcesStorage{
			records: map[uuid.UUID]*pipelineservice.Source{
				sourceID: {ID: sourceID, WorkspaceID: workspaceID},
			},
		},
		destinations: &createConnDestinationsStorage{
			records: map[uuid.UUID]*pipelineservice.Destination{
				destID: {ID: destID, WorkspaceID: workspaceID},
			},
		},
		connections: &createConnConnectionsStorage{
			records: map[uuid.UUID]*pipelineservice.Connection{},
		},
		connStates: &createConnConnectionStatesStorage{
			records: map[uuid.UUID]*pipelineservice.ConnectionState{},
		},
		streamGens: &createConnStreamGenerationsStorage{
			records: map[string]*pipelineservice.StreamGeneration{},
		},
		fake: pipelinefake.New(),
	}
}

func (s *createConnStore) Sources() pipelineservice.SourcesStorage {
	return s.sources
}

func (s *createConnStore) Destinations() pipelineservice.DestinationsStorage {
	return s.destinations
}

func (s *createConnStore) Connections() pipelineservice.ConnectionsStorage {
	return s.connections
}

func (s *createConnStore) ConnectionStates() pipelineservice.ConnectionStatesStorage {
	return s.connStates
}

func (s *createConnStore) StreamGenerations() pipelineservice.StreamGenerationsStorage {
	return s.streamGens
}

type createConnSourcesStorage struct {
	pipelineservice.SourcesStorage
	records map[uuid.UUID]*pipelineservice.Source
}

func (s *createConnSourcesStorage) First(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	id := f.ID.(*filter.EqualsFilter[uuid.UUID]).Value

	rec, ok := s.records[id]
	if !ok {
		return nil, pipelineservice.ErrSourceNotFound
	}

	cp := *rec

	return &cp, nil
}

type createConnDestinationsStorage struct {
	pipelineservice.DestinationsStorage
	records map[uuid.UUID]*pipelineservice.Destination
}

func (s *createConnDestinationsStorage) First(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Destination, error) {
	id := f.ID.(*filter.EqualsFilter[uuid.UUID]).Value

	rec, ok := s.records[id]
	if !ok {
		return nil, pipelineservice.ErrDestinationNotFound
	}

	cp := *rec

	return &cp, nil
}

type createConnConnectionsStorage struct {
	pipelineservice.ConnectionsStorage
	records map[uuid.UUID]*pipelineservice.Connection
}

func (s *createConnConnectionsStorage) Create(_ context.Context, conn *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	cp := *conn
	s.records[conn.ID] = &cp

	return &cp, nil
}

type createConnConnectionStatesStorage struct {
	pipelineservice.ConnectionStatesStorage
	records map[uuid.UUID]*pipelineservice.ConnectionState
}

type createConnStreamGenerationsStorage struct {
	pipelineservice.StreamGenerationsStorage
	records map[string]*pipelineservice.StreamGeneration
}

func TestCreateConnection_StoresNormalizedTags(t *testing.T) {
	workspaceID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	store := newCreateConnStore(workspaceID, sourceID, destID)

	uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{})
	created, err := uc.Execute(context.Background(), pipelineconnections.CreateConnectionParams{
		WorkspaceID:   workspaceID,
		Name:          "tagged",
		SourceID:      sourceID,
		DestinationID: destID,
		Tags:          []string{" Prod", "prod", "finance"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"finance", "prod"}, created.Tags)
	assert.Equal(t, []string{"finance", "prod"}, store.connections.records[created.ID].Tags)
}

func TestCreateConnection_InvalidTagIsValidationError(t *testing.T) {
	workspaceID := uuid.New()
	sourceID := uuid.New()
	destID := uuid.New()
	store := newCreateConnStore(workspaceID, sourceID, destID)

	uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{})
	_, err := uc.Execute(context.Background(), pipelineconnections.CreateConnectionParams{
		WorkspaceID:   workspaceID,
		Name:          "bad",
		SourceID:      sourceID,
		DestinationID: destID,
		Tags:          []string{"fin ance"},
	})

	var validationErr *pipelineservice.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "tags", validationErr.Field)
	assert.Contains(t, err.Error(), `"fin ance"`)
	assert.Empty(t, store.connections.records)
}

func seedDiscovery(t *testing.T, store *createConnStore, sourceID uuid.UUID, streams ...string) {
	t.Helper()

	catalog := protocol.AirbyteCatalog{}
	for _, name := range streams {
		catalog.Streams = append(catalog.Streams, protocol.AirbyteStream{
			Name:               name,
			JSONSchema:         json.RawMessage(`{"properties":{"id":{"type":"integer"}}}`),
			SupportedSyncModes: []protocol.SyncMode{protocol.SyncModeFullRefresh},
		})
	}

	raw, err := json.Marshal(catalog)
	require.NoError(t, err)

	store.fake.CatalogDiscoveriesTable.Insert(&pipelineservice.CatalogDiscovery{
		ID:          uuid.New(),
		SourceID:    sourceID,
		Version:     1,
		CatalogJSON: string(raw),
	})
}

func fullRefresh(name string) protocol.ConfiguredAirbyteStream {
	return protocol.ConfiguredAirbyteStream{
		Stream:              protocol.AirbyteStream{Name: name},
		SyncMode:            protocol.SyncModeFullRefresh,
		DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
	}
}

func TestCreateConnection_Streams(t *testing.T) {
	workspaceID, sourceID, destID := uuid.New(), uuid.New(), uuid.New()

	create := func(store *createConnStore, streams ...protocol.ConfiguredAirbyteStream) (*pipelineservice.Connection, error) {
		uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, pipelineservice.NoopAuditRecorder{})

		return uc.Execute(context.Background(), pipelineconnections.CreateConnectionParams{
			WorkspaceID:   workspaceID,
			Name:          "pg-to-bq",
			SourceID:      sourceID,
			DestinationID: destID,
			Streams:       streams,
		})
	}

	t.Run("configures the given streams with the connection", func(t *testing.T) {
		store := newCreateConnStore(workspaceID, sourceID, destID)
		seedDiscovery(t, store, sourceID, "users", "orders")

		created, err := create(store, fullRefresh("users"))
		require.NoError(t, err)

		catalogs := store.fake.ConfiguredCatalogsTable.All()
		require.Len(t, catalogs, 1)
		assert.Equal(t, created.ID, catalogs[0].ConnectionID)

		var stored []protocol.ConfiguredAirbyteStream
		require.NoError(t, json.Unmarshal([]byte(catalogs[0].StreamsJSON), &stored))
		require.Len(t, stored, 1)
		assert.Equal(t, "users", stored[0].Stream.Name)
		assert.JSONEq(t, `{"properties":{"id":{"type":"integer"}}}`, string(stored[0].Stream.JSONSchema),
			"the stream is enriched from the discovered catalog")
	})

	t.Run("catalog not discovered yet creates nothing", func(t *testing.T) {
		store := newCreateConnStore(workspaceID, sourceID, destID)

		_, err := create(store, fullRefresh("users"))
		require.ErrorIs(t, err, pipelineservice.ErrCatalogNotReady)
		assert.Contains(t, err.Error(), "retry")
		assert.Empty(t, store.connections.records)
		assert.Empty(t, store.fake.ConfiguredCatalogsTable.All())
	})

	t.Run("invalid stream creates nothing", func(t *testing.T) {
		store := newCreateConnStore(workspaceID, sourceID, destID)
		seedDiscovery(t, store, sourceID, "users")

		_, err := create(store, fullRefresh("missing"))

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Empty(t, store.connections.records)
		assert.Empty(t, store.fake.ConfiguredCatalogsTable.All())
	})

	t.Run("without streams the connection is created unconfigured", func(t *testing.T) {
		store := newCreateConnStore(workspaceID, sourceID, destID)

		created, err := create(store)
		require.NoError(t, err)
		assert.Contains(t, store.connections.records, created.ID)
		assert.Empty(t, store.fake.ConfiguredCatalogsTable.All())
	})
}
