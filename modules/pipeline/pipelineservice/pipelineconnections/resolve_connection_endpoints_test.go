package pipelineconnections_test

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

type endpointsStore struct {
	pipelineservice.Storage
	sources      *endpointsSources
	destinations *endpointsDestinations
	connectors   *endpointsConnectors
}

func (s *endpointsStore) Sources() pipelineservice.SourcesStorage { return s.sources }

func (s *endpointsStore) Destinations() pipelineservice.DestinationsStorage { return s.destinations }

func (s *endpointsStore) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return s.connectors
}

type endpointsSources struct {
	pipelineservice.SourcesStorage
	records []*pipelineservice.Source
	calls   int
}

func (s *endpointsSources) Find(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	s.calls++

	var out []*pipelineservice.Source

	for _, rec := range s.records {
		if matchesUUID(f.ID, rec.ID) && matchesUUID(f.WorkspaceID, rec.WorkspaceID) {
			out = append(out, rec)
		}
	}

	return out, nil
}

type endpointsDestinations struct {
	pipelineservice.DestinationsStorage
	records []*pipelineservice.Destination
	calls   int
}

func (s *endpointsDestinations) Find(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	s.calls++

	var out []*pipelineservice.Destination

	for _, rec := range s.records {
		if matchesUUID(f.ID, rec.ID) && matchesUUID(f.WorkspaceID, rec.WorkspaceID) {
			out = append(out, rec)
		}
	}

	return out, nil
}

type endpointsConnectors struct {
	pipelineservice.ManagedConnectorsStorage
	records []*pipelineservice.ManagedConnector
	calls   int
}

func (s *endpointsConnectors) Find(_ context.Context, f *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ManagedConnector, error) {
	s.calls++

	var out []*pipelineservice.ManagedConnector

	for _, rec := range s.records {
		if matchesUUID(f.ID, rec.ID) {
			out = append(out, rec)
		}
	}

	return out, nil
}

func TestResolveConnectionEndpoints(t *testing.T) {
	ws := uuid.New()
	otherWs := uuid.New()
	pgConnector := &pipelineservice.ManagedConnector{ID: uuid.New(), WorkspaceID: ws, Name: "Postgres"}
	bqConnector := &pipelineservice.ManagedConnector{ID: uuid.New(), WorkspaceID: ws, Name: "BigQuery"}
	src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, Name: "Orders DB", ManagedConnectorID: pgConnector.ID}
	foreignSrc := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: otherWs, Name: "Foreign", ManagedConnectorID: pgConnector.ID}
	dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, Name: "Warehouse", ManagedConnectorID: bqConnector.ID}

	newStore := func() *endpointsStore {
		return &endpointsStore{
			sources:      &endpointsSources{records: []*pipelineservice.Source{src, foreignSrc}},
			destinations: &endpointsDestinations{records: []*pipelineservice.Destination{dst}},
			connectors:   &endpointsConnectors{records: []*pipelineservice.ManagedConnector{pgConnector, bqConnector}},
		}
	}

	t.Run("resolves names with one lookup per entity type", func(t *testing.T) {
		store := newStore()
		first := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: src.ID, DestinationID: dst.ID}
		second := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: src.ID, DestinationID: dst.ID}

		got, err := pipelineconnections.NewResolveConnectionEndpoints(store).Execute(context.Background(), pipelineconnections.ResolveConnectionEndpointsParams{
			WorkspaceID: ws,
			Connections: []*pipelineservice.Connection{first, second},
		})
		require.NoError(t, err)

		want := pipelineconnections.ConnectionEndpoints{
			SourceName:               "Orders DB",
			SourceConnectorName:      "Postgres",
			DestinationName:          "Warehouse",
			DestinationConnectorName: "BigQuery",
		}
		assert.Equal(t, want, got[first.ID])
		assert.Equal(t, want, got[second.ID])
		assert.Equal(t, 1, store.sources.calls)
		assert.Equal(t, 1, store.destinations.calls)
		assert.Equal(t, 1, store.connectors.calls)
	})

	t.Run("leaves names empty for records outside the workspace", func(t *testing.T) {
		store := newStore()
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: foreignSrc.ID, DestinationID: uuid.New()}

		got, err := pipelineconnections.NewResolveConnectionEndpoints(store).Execute(context.Background(), pipelineconnections.ResolveConnectionEndpointsParams{
			WorkspaceID: ws,
			Connections: []*pipelineservice.Connection{conn},
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineconnections.ConnectionEndpoints{}, got[conn.ID])
		assert.Equal(t, 0, store.connectors.calls)
	})

	t.Run("skips lookups for an empty page", func(t *testing.T) {
		store := newStore()

		got, err := pipelineconnections.NewResolveConnectionEndpoints(store).Execute(context.Background(), pipelineconnections.ResolveConnectionEndpointsParams{WorkspaceID: ws})
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, 0, store.sources.calls)
		assert.Equal(t, 0, store.destinations.calls)
	})
}
