package pipelineconfig_test

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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconfig"
)

type fakeStore struct {
	pipelineservice.Storage

	sources      *fakeSources
	destinations *fakeDestinations
	connections  *fakeConnections
}

func (s *fakeStore) Sources() pipelineservice.SourcesStorage           { return s.sources }
func (s *fakeStore) Destinations() pipelineservice.DestinationsStorage { return s.destinations }
func (s *fakeStore) Connections() pipelineservice.ConnectionsStorage   { return s.connections }

func equals[T comparable](f filter.Filter[T], v T) bool {
	if f == nil {
		return true
	}

	return f.(*filter.EqualsFilter[T]).Value == v
}

type fakeSources struct {
	pipelineservice.SourcesStorage
	records []*pipelineservice.Source
}

func (s *fakeSources) Find(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	var out []*pipelineservice.Source

	for _, rec := range s.records {
		if equals(f.WorkspaceID, rec.WorkspaceID) && equals(f.Name, rec.Name) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeSources) Create(_ context.Context, rec *pipelineservice.Source) (*pipelineservice.Source, error) {
	s.records = append(s.records, rec)

	return rec, nil
}

func (s *fakeSources) Update(_ context.Context, rec *pipelineservice.Source) (*pipelineservice.Source, error) {
	return rec, nil
}

type fakeDestinations struct {
	pipelineservice.DestinationsStorage
	records []*pipelineservice.Destination
}

func (s *fakeDestinations) Find(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	var out []*pipelineservice.Destination

	for _, rec := range s.records {
		if equals(f.WorkspaceID, rec.WorkspaceID) && equals(f.Name, rec.Name) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeDestinations) Create(_ context.Context, rec *pipelineservice.Destination) (*pipelineservice.Destination, error) {
	s.records = append(s.records, rec)

	return rec, nil
}

type fakeConnections struct {
	pipelineservice.ConnectionsStorage
	records []*pipelineservice.Connection
}

func (s *fakeConnections) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	var out []*pipelineservice.Connection

	for _, rec := range s.records {
		if equals(f.WorkspaceID, rec.WorkspaceID) && equals(f.SourceID, rec.SourceID) && equals(f.DestinationID, rec.DestinationID) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeConnections) Create(_ context.Context, rec *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	s.records = append(s.records, rec)

	return rec, nil
}

func (s *fakeConnections) Update(_ context.Context, rec *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	return rec, nil
}

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (r *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}

func TestImportConfig_RecordsAuditEvents(t *testing.T) {
	workspaceID := uuid.New()
	oldConnectorID := uuid.New()
	newConnectorID := uuid.New()
	destConnectorID := uuid.New()

	existingSource := &pipelineservice.Source{
		ID:                 uuid.New(),
		WorkspaceID:        workspaceID,
		Name:               "pg",
		ManagedConnectorID: oldConnectorID,
		Config:             `{"password":"hunter2"}`,
	}
	schedule := "0 * * * *"
	store := &fakeStore{
		sources:      &fakeSources{records: []*pipelineservice.Source{existingSource}},
		destinations: &fakeDestinations{},
		connections:  &fakeConnections{},
	}
	audit := &recordingAudit{}
	uc := pipelineconfig.NewImportConfig(store, pipelineservice.Config{DefaultMaxAttempts: 3}, audit)

	result, err := uc.Execute(context.Background(), pipelineconfig.ImportConfigParams{
		WorkspaceID: workspaceID,
		ConfigYAML: []byte(`version: "1"
sources:
  - name: pg
    managed_connector_id: ` + newConnectorID.String() + `
    config:
      password: "********"
destinations:
  - name: bq
    managed_connector_id: ` + destConnectorID.String() + `
    config:
      credentials_json: top-secret
connections:
  - source: pg
    destination: bq
    schedule: "` + schedule + `"
    enabled: true
`),
	})
	require.NoError(t, err)
	require.Empty(t, result.Errors)
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 1, result.Updated)

	require.Len(t, store.destinations.records, 1)
	require.Len(t, store.connections.records, 1)
	require.Len(t, audit.events, 4)

	assert.Equal(t, pipelineservice.AuditEvent{
		WorkspaceID:   workspaceID,
		Action:        "update",
		ResourceType:  "source",
		ResourceID:    existingSource.ID,
		ResourceLabel: "pg",
		Before:        map[string]any{"name": "pg", "managed_connector_id": oldConnectorID.String()},
		After:         map[string]any{"name": "pg", "managed_connector_id": newConnectorID.String()},
	}, audit.events[0])

	dst := store.destinations.records[0]
	assert.Equal(t, pipelineservice.AuditEvent{
		WorkspaceID:   workspaceID,
		Action:        "create",
		ResourceType:  "destination",
		ResourceID:    dst.ID,
		ResourceLabel: "bq",
		After:         map[string]any{"name": "bq", "managed_connector_id": destConnectorID.String()},
	}, audit.events[1])

	conn := store.connections.records[0]
	assert.Equal(t, pipelineservice.AuditEvent{
		WorkspaceID:   workspaceID,
		Action:        "create",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: "pg -> bq",
		After: map[string]any{
			"name":         "pg -> bq",
			"schedule":     schedule,
			"status":       pipelineservice.ConnectionStatusActive.String(),
			"max_attempts": 3,
		},
	}, audit.events[2])

	assert.Equal(t, pipelineservice.AuditEvent{
		WorkspaceID:   workspaceID,
		Action:        "update",
		ResourceType:  "workspace",
		ResourceID:    workspaceID,
		ResourceLabel: "config import",
		After:         map[string]any{"sources": 1, "destinations": 1, "connections": 1},
	}, audit.events[3])
}

func TestImportConfig_UpdatedConnectionRecordsBeforeAndAfter(t *testing.T) {
	workspaceID := uuid.New()
	src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: workspaceID, Name: "pg"}
	dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: workspaceID, Name: "bq"}
	oldSchedule := "0 0 * * *"
	conn := &pipelineservice.Connection{
		ID:            uuid.New(),
		WorkspaceID:   workspaceID,
		Name:          "nightly",
		Status:        pipelineservice.ConnectionStatusActive,
		SourceID:      src.ID,
		DestinationID: dst.ID,
		Schedule:      &oldSchedule,
		MaxAttempts:   3,
	}
	store := &fakeStore{
		sources:      &fakeSources{records: []*pipelineservice.Source{src}},
		destinations: &fakeDestinations{records: []*pipelineservice.Destination{dst}},
		connections:  &fakeConnections{records: []*pipelineservice.Connection{conn}},
	}
	audit := &recordingAudit{}
	uc := pipelineconfig.NewImportConfig(store, pipelineservice.Config{}, audit)

	result, err := uc.Execute(context.Background(), pipelineconfig.ImportConfigParams{
		WorkspaceID: workspaceID,
		ConfigYAML: []byte(`version: "1"
connections:
  - source: pg
    destination: bq
    max_attempts: 5
    enabled: false
`),
	})
	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Len(t, audit.events, 2)

	assert.Equal(t, pipelineservice.AuditEvent{
		WorkspaceID:   workspaceID,
		Action:        "update",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: "nightly",
		Before: map[string]any{
			"name":         "nightly",
			"schedule":     oldSchedule,
			"status":       pipelineservice.ConnectionStatusActive.String(),
			"max_attempts": 3,
		},
		After: map[string]any{
			"name":         "nightly",
			"schedule":     nil,
			"status":       pipelineservice.ConnectionStatusInactive.String(),
			"max_attempts": 5,
		},
	}, audit.events[0])

	assert.Equal(t, map[string]any{"sources": 0, "destinations": 0, "connections": 1}, audit.events[1].After)
}
