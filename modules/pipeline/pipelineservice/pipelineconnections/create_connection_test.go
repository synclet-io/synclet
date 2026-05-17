package pipelineconnections_test

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
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

	uc := pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3})
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

// ---- minimal in-memory storage scaffolding for the CreateConnection use case ----

type createConnStore struct {
	pipelineservice.Storage
	sources      *createConnSourcesStorage
	destinations *createConnDestinationsStorage
	connections  *createConnConnectionsStorage
	connStates   *createConnConnectionStatesStorage
	streamGens   *createConnStreamGenerationsStorage
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
