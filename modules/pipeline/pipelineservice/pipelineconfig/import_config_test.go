package pipelineconfig_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconfig"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

const passwordSpec = `{"connectionSpecification":{"properties":{"host":{"type":"string"},"password":{"type":"string","airbyte_secret":true}}}}`

type fakeStore struct {
	pipelineservice.Storage

	sources      *fakeSources
	destinations *fakeDestinations
	connections  *fakeConnections
	connectors   *fakeConnectors
	findErr      error
}

func (s *fakeStore) Sources() pipelineservice.SourcesStorage           { return s.sources }
func (s *fakeStore) Destinations() pipelineservice.DestinationsStorage { return s.destinations }
func (s *fakeStore) Connections() pipelineservice.ConnectionsStorage   { return s.connections }
func (s *fakeStore) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return s.connectors
}

func equals[T comparable](f filter.Filter[T], v T) bool {
	if f == nil {
		return true
	}

	return f.(*filter.EqualsFilter[T]).Value == v
}

type fakeSources struct {
	pipelineservice.SourcesStorage
	store   *fakeStore
	records []*pipelineservice.Source
}

func (s *fakeSources) match(f *pipelineservice.SourceFilter, rec *pipelineservice.Source) bool {
	return equals(f.ID, rec.ID) && equals(f.WorkspaceID, rec.WorkspaceID) && equals(f.Name, rec.Name)
}

func (s *fakeSources) Find(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	if s.store.findErr != nil {
		return nil, s.store.findErr
	}

	var out []*pipelineservice.Source

	for _, rec := range s.records {
		if s.match(f, rec) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeSources) First(ctx context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	found, _ := s.Find(ctx, f)
	if len(found) == 0 {
		return nil, pipelineservice.NotFoundError("Source")
	}

	return found[0], nil
}

func (s *fakeSources) Create(_ context.Context, rec *pipelineservice.Source) (*pipelineservice.Source, error) {
	s.records = append(s.records, rec)

	return rec, nil
}

func (s *fakeSources) Update(_ context.Context, rec *pipelineservice.Source) (*pipelineservice.Source, error) {
	for i, existing := range s.records {
		if existing.ID == rec.ID {
			s.records[i] = rec
		}
	}

	return rec, nil
}

type fakeDestinations struct {
	pipelineservice.DestinationsStorage
	records []*pipelineservice.Destination
}

func (s *fakeDestinations) Find(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	var out []*pipelineservice.Destination

	for _, rec := range s.records {
		if equals(f.ID, rec.ID) && equals(f.WorkspaceID, rec.WorkspaceID) && equals(f.Name, rec.Name) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeDestinations) First(ctx context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Destination, error) {
	found, _ := s.Find(ctx, f)
	if len(found) == 0 {
		return nil, pipelineservice.NotFoundError("Destination")
	}

	return found[0], nil
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
		if equals(f.ID, rec.ID) && equals(f.WorkspaceID, rec.WorkspaceID) &&
			equals(f.SourceID, rec.SourceID) && equals(f.DestinationID, rec.DestinationID) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *fakeConnections) First(ctx context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	found, _ := s.Find(ctx, f)
	if len(found) == 0 {
		return nil, pipelineservice.NotFoundError("Connection")
	}

	return found[0], nil
}

func (s *fakeConnections) Create(_ context.Context, rec *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	s.records = append(s.records, rec)

	return rec, nil
}

func (s *fakeConnections) Update(_ context.Context, rec *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	for i, existing := range s.records {
		if existing.ID == rec.ID {
			s.records[i] = rec
		}
	}

	return rec, nil
}

type fakeConnectors struct {
	pipelineservice.ManagedConnectorsStorage
	records []*pipelineservice.ManagedConnector
}

func (s *fakeConnectors) First(_ context.Context, f *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	for _, rec := range s.records {
		if equals(f.ID, rec.ID) && equals(f.WorkspaceID, rec.WorkspaceID) {
			return rec, nil
		}
	}

	return nil, pipelineservice.NotFoundError("ManagedConnector")
}

type fakeSecrets struct {
	values map[string]string
}

func (s *fakeSecrets) StoreSecret(_ context.Context, _ string, _ uuid.UUID, plaintext string) (string, error) {
	ref := secretutil.SecretRefPrefix + uuid.NewString()
	s.values[ref] = plaintext

	return ref, nil
}

func (s *fakeSecrets) RetrieveSecret(_ context.Context, ref string) (string, error) {
	return s.values[ref], nil
}

func (s *fakeSecrets) DeleteSecret(_ context.Context, ref string) error {
	delete(s.values, ref)

	return nil
}

func (s *fakeSecrets) DeleteByOwner(context.Context, string, uuid.UUID) error { return nil }

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (r *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}

func (r *recordingAudit) find(action, resourceType string) []pipelineservice.AuditEvent {
	var out []pipelineservice.AuditEvent

	for _, event := range r.events {
		if event.Action == action && event.ResourceType == resourceType {
			out = append(out, event)
		}
	}

	return out
}

type noopDelegate struct{}

func (noopDelegate) Info(context.Context, string, ...interface{})         {}
func (noopDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopDelegate) Named(string) logging.Delegate                      { return d }
func (d noopDelegate) SkipCallers(int) logging.Delegate                   { return d }

type fixture struct {
	workspaceID uuid.UUID
	connector   *pipelineservice.ManagedConnector
	store       *fakeStore
	secrets     *fakeSecrets
	audit       *recordingAudit
	uc          *pipelineconfig.ImportConfig
}

func newFixture() *fixture {
	workspaceID := uuid.New()
	connector := &pipelineservice.ManagedConnector{ID: uuid.New(), WorkspaceID: workspaceID, Spec: passwordSpec}
	store := &fakeStore{
		destinations: &fakeDestinations{},
		connections:  &fakeConnections{},
		connectors:   &fakeConnectors{records: []*pipelineservice.ManagedConnector{connector}},
	}
	store.sources = &fakeSources{store: store}
	secrets := &fakeSecrets{values: map[string]string{}}
	audit := &recordingAudit{}
	logger := &logging.Logger{Delegate: noopDelegate{}}
	getConnection := pipelineconnections.NewGetConnection(store)
	updateStatus := pipelineconnections.NewUpdateConnectionStatus(store)

	return &fixture{
		workspaceID: workspaceID,
		connector:   connector,
		store:       store,
		secrets:     secrets,
		audit:       audit,
		uc: pipelineconfig.NewImportConfig(
			store,
			audit,
			pipelinesources.NewCreateSource(store, secrets, audit, logger),
			pipelinesources.NewUpdateSource(store, secrets, audit, logger),
			pipelinedestinations.NewCreateDestination(store, secrets, audit, logger),
			pipelinedestinations.NewUpdateDestination(store, secrets, audit, logger),
			pipelineconnections.NewCreateConnection(store, pipelineservice.Config{DefaultMaxAttempts: 3}, audit),
			pipelineconnections.NewUpdateConnection(store, audit),
			pipelineconnections.NewEnableConnection(getConnection, updateStatus, audit),
			pipelineconnections.NewDisableConnection(getConnection, updateStatus, audit),
		),
	}
}

func (f *fixture) run(t *testing.T, yaml string) *pipelineconfig.ImportConfigResult {
	t.Helper()

	result, err := f.uc.Execute(context.Background(), pipelineconfig.ImportConfigParams{
		WorkspaceID: f.workspaceID,
		ConfigYAML:  []byte(yaml),
	})
	require.NoError(t, err)

	return result
}

func configOf(t *testing.T, raw string) map[string]any {
	t.Helper()

	var parsed map[string]any

	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	return parsed
}

func TestImportConfig_CreateEncryptsSecretsAndDropsPlaceholders(t *testing.T) {
	f := newFixture()

	result := f.run(t, `version: "1"
sources:
  - name: pg
    managed_connector_id: `+f.connector.ID.String()+`
    config:
      host: db
      password: hunter2
      token: "***REDACTED***"
      stolen: "$secret:`+uuid.NewString()+`"
`)
	require.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Created)

	require.Len(t, f.store.sources.records, 1)
	stored := configOf(t, f.store.sources.records[0].Config)
	assert.Equal(t, "db", stored["host"])
	assert.NotContains(t, stored, "token")
	assert.NotContains(t, stored, "stolen")

	ref, ok := stored["password"].(string)
	require.True(t, ok)
	require.True(t, secretutil.IsSecretRef(ref), "password stored as %q", ref)
	assert.Equal(t, "hunter2", f.secrets.values[ref])

	assert.Len(t, f.audit.find("create", "source"), 1)
	assert.Equal(t, map[string]any{"sources": 1, "destinations": 0, "connections": 0}, f.audit.find("update", "workspace")[0].After)
}

func TestImportConfig_UpdateKeepsRedactedSecretsAndUnlistedFields(t *testing.T) {
	f := newFixture()
	ref := secretutil.SecretRefPrefix + uuid.NewString()
	f.secrets.values[ref] = "hunter2"
	existing := &pipelineservice.Source{
		ID:                 uuid.New(),
		WorkspaceID:        f.workspaceID,
		Name:               "pg",
		ManagedConnectorID: f.connector.ID,
		Config:             `{"host":"old","port":5432,"password":"` + ref + `"}`,
	}
	f.store.sources.records = []*pipelineservice.Source{existing}

	result := f.run(t, `version: "1"
sources:
  - name: pg
    managed_connector_id: `+f.connector.ID.String()+`
    config:
      host: new
      password: "***REDACTED***"
`)
	require.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Updated)

	stored := configOf(t, f.store.sources.records[0].Config)
	assert.Equal(t, map[string]any{"host": "new", "port": float64(5432), "password": ref}, stored)
	assert.Len(t, f.audit.find("update", "source"), 1)
}

func TestImportConfig_RejectsManagedConnectorChange(t *testing.T) {
	f := newFixture()
	f.store.sources.records = []*pipelineservice.Source{{
		ID:                 uuid.New(),
		WorkspaceID:        f.workspaceID,
		Name:               "pg",
		ManagedConnectorID: f.connector.ID,
		Config:             `{}`,
	}}

	result := f.run(t, `version: "1"
sources:
  - name: pg
    managed_connector_id: `+uuid.NewString()+`
`)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "managed_connector_id cannot be changed")
	assert.Zero(t, result.Updated)
}

func TestImportConfig_ConnectorFromAnotherWorkspaceIsAnEntryError(t *testing.T) {
	f := newFixture()
	foreign := &pipelineservice.ManagedConnector{ID: uuid.New(), WorkspaceID: uuid.New(), Spec: passwordSpec}
	f.store.connectors.records = append(f.store.connectors.records, foreign)

	result := f.run(t, `version: "1"
destinations:
  - name: bq
    managed_connector_id: `+foreign.ID.String()+`
`)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], `creating destination "bq"`)
	assert.Empty(t, f.store.destinations.records)
}

func TestImportConfig_InvalidCronIsAnEntryError(t *testing.T) {
	f := newFixture()
	f.store.sources.records = []*pipelineservice.Source{{ID: uuid.New(), WorkspaceID: f.workspaceID, Name: "pg"}}
	f.store.destinations.records = []*pipelineservice.Destination{{ID: uuid.New(), WorkspaceID: f.workspaceID, Name: "bq"}}

	result := f.run(t, `version: "1"
connections:
  - source: pg
    destination: bq
    schedule: "every hour"
    enabled: true
`)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "invalid cron expression")
	assert.Empty(t, f.store.connections.records)
}

func TestImportConfig_CreatesDisabledConnection(t *testing.T) {
	f := newFixture()
	f.store.sources.records = []*pipelineservice.Source{{ID: uuid.New(), WorkspaceID: f.workspaceID, Name: "pg"}}
	f.store.destinations.records = []*pipelineservice.Destination{{ID: uuid.New(), WorkspaceID: f.workspaceID, Name: "bq"}}

	result := f.run(t, `version: "1"
connections:
  - source: pg
    destination: bq
    schedule: "0 * * * *"
    enabled: false
`)
	require.Empty(t, result.Errors)
	require.Len(t, f.store.connections.records, 1)

	conn := f.store.connections.records[0]
	assert.Equal(t, pipelineservice.ConnectionStatusInactive, conn.Status)
	assert.Nil(t, conn.NextScheduledAt)
	assert.Equal(t, 3, conn.MaxAttempts)
	assert.Equal(t, pipelineservice.SchemaChangePolicyPropagate, conn.SchemaChangePolicy)
	assert.Len(t, f.audit.find("create", "connection"), 1)
}

func TestImportConfig_UpdatesConnectionThroughUseCases(t *testing.T) {
	f := newFixture()
	src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: f.workspaceID, Name: "pg"}
	dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: f.workspaceID, Name: "bq"}
	oldSchedule := "0 0 * * *"
	conn := &pipelineservice.Connection{
		ID:            uuid.New(),
		WorkspaceID:   f.workspaceID,
		Name:          "nightly",
		Status:        pipelineservice.ConnectionStatusActive,
		SourceID:      src.ID,
		DestinationID: dst.ID,
		Schedule:      &oldSchedule,
		MaxAttempts:   3,
	}
	f.store.sources.records = []*pipelineservice.Source{src}
	f.store.destinations.records = []*pipelineservice.Destination{dst}
	f.store.connections.records = []*pipelineservice.Connection{conn}

	result := f.run(t, `version: "1"
connections:
  - source: pg
    destination: bq
    max_attempts: 5
    enabled: false
`)
	require.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Updated)

	updated := f.store.connections.records[0]
	assert.Nil(t, updated.Schedule)
	assert.Equal(t, 5, updated.MaxAttempts)
	assert.Equal(t, pipelineservice.ConnectionStatusInactive, updated.Status)
	assert.Len(t, f.audit.find("update", "connection"), 1)
	assert.Len(t, f.audit.find("disable", "connection"), 1)
	assert.Equal(t, map[string]any{"sources": 0, "destinations": 0, "connections": 1}, f.audit.find("update", "workspace")[0].After)
}

func TestImportConfig_LookupErrorAbortsImport(t *testing.T) {
	f := newFixture()
	f.store.findErr = errors.New("connection refused")

	_, err := f.uc.Execute(context.Background(), pipelineconfig.ImportConfigParams{
		WorkspaceID: f.workspaceID,
		ConfigYAML: []byte(`version: "1"
sources:
  - name: pg
    managed_connector_id: ` + f.connector.ID.String() + `
`),
	})
	require.ErrorContains(t, err, "connection refused")
	assert.Empty(t, f.audit.events)
}

func TestImportConfig_InvalidDocumentIsValidationError(t *testing.T) {
	f := newFixture()

	for _, doc := range []string{"version: [", `version: "2"`} {
		_, err := f.uc.Execute(context.Background(), pipelineconfig.ImportConfigParams{
			WorkspaceID: f.workspaceID,
			ConfigYAML:  []byte(doc),
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr, doc)
	}
}
