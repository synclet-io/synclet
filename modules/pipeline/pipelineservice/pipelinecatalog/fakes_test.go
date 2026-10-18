package pipelinecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

var errStorage = errors.New("storage unavailable")

const nestedSchema = `{
	"type": "object",
	"properties": {
		"id": {"type": "integer"},
		"name": {"type": "string"},
		"updated_at": {"type": "string"},
		"address": {"type": "object", "properties": {"city": {"type": "string"}}}
	}
}`

func fields(paths ...[]string) []protocol.SelectedField {
	out := make([]protocol.SelectedField, len(paths))
	for i, p := range paths {
		out[i] = protocol.SelectedField{FieldPath: p}
	}

	return out
}

type fakeStore struct {
	pipelineservice.Storage

	connections *fakeConnections
	sources     *fakeSources
	discoveries *fakeDiscoveries
	configured  *fakeConfiguredCatalogs
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		connections: &fakeConnections{records: map[uuid.UUID]*pipelineservice.Connection{}},
		sources:     &fakeSources{records: map[uuid.UUID]*pipelineservice.Source{}},
		discoveries: &fakeDiscoveries{},
		configured:  &fakeConfiguredCatalogs{records: map[uuid.UUID]*pipelineservice.ConfiguredCatalog{}},
	}
}

func (s *fakeStore) Connections() pipelineservice.ConnectionsStorage { return s.connections }

func (s *fakeStore) Sources() pipelineservice.SourcesStorage { return s.sources }

func (s *fakeStore) CatalogDiscoveries() pipelineservice.CatalogDiscoveriesStorage {
	return s.discoveries
}

func (s *fakeStore) ConfiguredCatalogs() pipelineservice.ConfiguredCatalogsStorage {
	return s.configured
}

func equalsValue[T any](f filter.Filter[T]) (T, bool) {
	eq, ok := f.(*filter.EqualsFilter[T])
	if !ok {
		var zero T

		return zero, false
	}

	return eq.Value, true
}

type fakeConnections struct {
	pipelineservice.ConnectionsStorage

	records map[uuid.UUID]*pipelineservice.Connection
}

func (s *fakeConnections) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	for _, conn := range s.records {
		if id, ok := equalsValue(f.ID); ok && conn.ID != id {
			continue
		}

		if wid, ok := equalsValue(f.WorkspaceID); ok && conn.WorkspaceID != wid {
			continue
		}

		cp := *conn

		return &cp, nil
	}

	return nil, pipelineservice.ErrConnectionNotFound
}

type fakeSources struct {
	pipelineservice.SourcesStorage

	records map[uuid.UUID]*pipelineservice.Source
}

func (s *fakeSources) First(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	for _, src := range s.records {
		if id, ok := equalsValue(f.ID); ok && src.ID != id {
			continue
		}

		if wid, ok := equalsValue(f.WorkspaceID); ok && src.WorkspaceID != wid {
			continue
		}

		cp := *src

		return &cp, nil
	}

	return nil, pipelineservice.ErrSourceNotFound
}

type fakeDiscoveries struct {
	pipelineservice.CatalogDiscoveriesStorage

	records    []*pipelineservice.CatalogDiscovery
	firstErr   error
	createErr  error
	firstCalls int
}

// First returns the highest-version discovery for the filtered source.
func (s *fakeDiscoveries) First(_ context.Context, f *pipelineservice.CatalogDiscoveryFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.CatalogDiscovery, error) {
	s.firstCalls++

	if s.firstErr != nil {
		return nil, s.firstErr
	}

	var latest *pipelineservice.CatalogDiscovery

	for _, rec := range s.records {
		if sid, ok := equalsValue(f.SourceID); ok && rec.SourceID != sid {
			continue
		}

		if latest == nil || rec.Version > latest.Version {
			latest = rec
		}
	}

	if latest == nil {
		return nil, pipelineservice.ErrCatalogDiscoveryNotFound
	}

	cp := *latest

	return &cp, nil
}

func (s *fakeDiscoveries) Create(_ context.Context, rec *pipelineservice.CatalogDiscovery) (*pipelineservice.CatalogDiscovery, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}

	s.records = append(s.records, rec)

	return rec, nil
}

type fakeConfiguredCatalogs struct {
	pipelineservice.ConfiguredCatalogsStorage

	records   map[uuid.UUID]*pipelineservice.ConfiguredCatalog
	firstErr  error
	createErr error
	updateErr error
	created   int
	updated   int
}

func (s *fakeConfiguredCatalogs) First(_ context.Context, f *pipelineservice.ConfiguredCatalogFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConfiguredCatalog, error) {
	if s.firstErr != nil {
		return nil, s.firstErr
	}

	connID, _ := equalsValue(f.ConnectionID)

	rec, ok := s.records[connID]
	if !ok {
		return nil, pipelineservice.ErrConfiguredCatalogNotFound
	}

	cp := *rec

	return &cp, nil
}

func (s *fakeConfiguredCatalogs) Find(_ context.Context, f *pipelineservice.ConfiguredCatalogFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ConfiguredCatalog, error) {
	if s.firstErr != nil {
		return nil, s.firstErr
	}

	in, ok := f.ConnectionID.(*filter.InFilter[uuid.UUID])
	if !ok {
		return nil, errors.New("fake supports only ConnectionID In filter")
	}

	var out []*pipelineservice.ConfiguredCatalog

	for _, id := range in.Values {
		if rec, ok := s.records[id]; ok {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeConfiguredCatalogs) Create(_ context.Context, rec *pipelineservice.ConfiguredCatalog) (*pipelineservice.ConfiguredCatalog, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}

	s.created++
	s.records[rec.ConnectionID] = rec

	return rec, nil
}

func (s *fakeConfiguredCatalogs) Update(_ context.Context, rec *pipelineservice.ConfiguredCatalog) (*pipelineservice.ConfiguredCatalog, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}

	s.updated++
	s.records[rec.ConnectionID] = rec

	return rec, nil
}

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (r *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}

type fakeDiscoverer struct {
	catalog *protocol.AirbyteCatalog
	err     error
}

func (d *fakeDiscoverer) Discover(_ context.Context, _ string, _ json.RawMessage) (*protocol.AirbyteCatalog, error) {
	return d.catalog, d.err
}

func seedDiscovery(t *testing.T, store *fakeStore, sourceID uuid.UUID, version int, catalog protocol.AirbyteCatalog) {
	t.Helper()

	raw, err := json.Marshal(catalog)
	require.NoError(t, err)

	store.discoveries.records = append(store.discoveries.records, &pipelineservice.CatalogDiscovery{
		ID:          uuid.New(),
		SourceID:    sourceID,
		Version:     version,
		CatalogJSON: string(raw),
	})
}

func seedConfigured(t *testing.T, store *fakeStore, connectionID uuid.UUID, streams []protocol.ConfiguredAirbyteStream) {
	t.Helper()

	raw, err := json.Marshal(streams)
	require.NoError(t, err)

	store.configured.records[connectionID] = &pipelineservice.ConfiguredCatalog{
		ID:           uuid.New(),
		ConnectionID: connectionID,
		StreamsJSON:  string(raw),
	}
}

func storedStreams(t *testing.T, store *fakeStore, connectionID uuid.UUID) []protocol.ConfiguredAirbyteStream {
	t.Helper()

	rec, ok := store.configured.records[connectionID]
	require.True(t, ok, "configured catalog not stored")

	var streams []protocol.ConfiguredAirbyteStream
	require.NoError(t, json.Unmarshal([]byte(rec.StreamsJSON), &streams))

	return streams
}
