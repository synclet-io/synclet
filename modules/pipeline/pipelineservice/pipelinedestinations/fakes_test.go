package pipelinedestinations_test

import (
	"context"
	"errors"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

const (
	ownerType = "destination"

	specWithPassword = `{"connectionSpecification":{"type":"object","properties":{
		"host":{"type":"string"},
		"password":{"type":"string","airbyte_secret":true}}}}`

	specWithPatOnly = `{"connectionSpecification":{"type":"object","properties":{
		"host":{"type":"string"},
		"pat":{"type":"string","airbyte_secret":true},
		"password":{"type":"string"},
		"access_token":{"type":"string"}}}}`

	specWithoutSecrets = `{"connectionSpecification":{"type":"object","properties":{"host":{"type":"string"}}}}`
)

type destStore struct {
	pipelineservice.Storage
	destinations *destStorage
	connectors   *connectorStorage
	connections  *connStorage
}

func (s *destStore) Destinations() pipelineservice.DestinationsStorage {
	return s.destinations
}

func (s *destStore) Connections() pipelineservice.ConnectionsStorage {
	return s.connections
}

func (s *destStore) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return s.connectors
}

type destStorage struct {
	pipelineservice.DestinationsStorage
	records   map[uuid.UUID]*pipelineservice.Destination
	createErr error
	updateErr error
	deleteErr error
}

func matchesDestination(rec *pipelineservice.Destination, f *pipelineservice.DestinationFilter) bool {
	if f.ID != nil {
		if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.ID != eq.Value {
			return false
		}
	}

	if f.WorkspaceID != nil {
		if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.WorkspaceID != eq.Value {
			return false
		}
	}

	return true
}

func (s *destStorage) First(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Destination, error) {
	for _, rec := range s.records {
		if !matchesDestination(rec, f) {
			continue
		}

		cp := *rec

		return &cp, nil
	}

	return nil, pipelineservice.ErrDestinationNotFound
}

func (s *destStorage) Find(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	out := make([]*pipelineservice.Destination, 0, len(s.records))

	for _, rec := range s.records {
		if !matchesDestination(rec, f) {
			continue
		}

		cp := *rec
		out = append(out, &cp)
	}

	return out, nil
}

func (s *destStorage) Create(_ context.Context, model *pipelineservice.Destination) (*pipelineservice.Destination, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}

	cp := *model
	s.records[model.ID] = &cp

	return model, nil
}

func (s *destStorage) Update(_ context.Context, model *pipelineservice.Destination) (*pipelineservice.Destination, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}

	if _, ok := s.records[model.ID]; !ok {
		return nil, pipelineservice.ErrDestinationNotFound
	}

	cp := *model
	s.records[model.ID] = &cp

	return model, nil
}

func (s *destStorage) Delete(_ context.Context, f *pipelineservice.DestinationFilter) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}

	for id, rec := range s.records {
		if matchesDestination(rec, f) {
			delete(s.records, id)
		}
	}

	return nil
}

type connectorStorage struct {
	pipelineservice.ManagedConnectorsStorage
	records map[uuid.UUID]*pipelineservice.ManagedConnector
}

func (s *connectorStorage) First(_ context.Context, f *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	for _, rec := range s.records {
		if f.ID != nil {
			if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.ID != eq.Value {
				continue
			}
		}

		if f.WorkspaceID != nil {
			if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.WorkspaceID != eq.Value {
				continue
			}
		}

		cp := *rec

		return &cp, nil
	}

	return nil, pipelineservice.ErrManagedConnectorNotFound
}

func newDestStore(records ...*pipelineservice.Destination) *destStore {
	m := make(map[uuid.UUID]*pipelineservice.Destination, len(records))
	connectors := make(map[uuid.UUID]*pipelineservice.ManagedConnector, len(records))

	for _, r := range records {
		m[r.ID] = r
		connectors[r.ManagedConnectorID] = &pipelineservice.ManagedConnector{
			ID:            r.ManagedConnectorID,
			WorkspaceID:   r.WorkspaceID,
			Name:          "connector",
			ConnectorType: pipelineservice.ConnectorTypeDestination,
			Spec:          "{}",
		}
	}

	return &destStore{
		destinations: &destStorage{records: m},
		connectors:   &connectorStorage{records: connectors},
		connections:  &connStorage{},
	}
}

func (s *destStore) addConnector(workspaceID uuid.UUID, spec string) *pipelineservice.ManagedConnector {
	conn := &pipelineservice.ManagedConnector{
		ID:            uuid.New(),
		WorkspaceID:   workspaceID,
		Name:          "connector",
		ConnectorType: pipelineservice.ConnectorTypeDestination,
		Spec:          spec,
	}
	s.connectors.records[conn.ID] = conn

	return conn
}

type secretOwner struct {
	ownerType string
	ownerID   uuid.UUID
}

type fakeSecrets struct {
	values           map[string]string
	owners           map[string]secretOwner
	storeErr         error
	deleteErr        error
	deleteByOwnerErr error
	deletedRefs      []string
	deletedOwners    []secretOwner
}

func newFakeSecrets() *fakeSecrets {
	return &fakeSecrets{values: map[string]string{}, owners: map[string]secretOwner{}}
}

func (f *fakeSecrets) StoreSecret(_ context.Context, ownerType string, ownerID uuid.UUID, plaintext string) (string, error) {
	if f.storeErr != nil {
		return "", f.storeErr
	}

	ref := secretutil.MakeSecretRef(uuid.New())
	f.values[ref] = plaintext
	f.owners[ref] = secretOwner{ownerType: ownerType, ownerID: ownerID}

	return ref, nil
}

func (f *fakeSecrets) RetrieveSecret(_ context.Context, secretRef string) (string, error) {
	plaintext, ok := f.values[secretRef]
	if !ok {
		return "", errors.New("secret not found")
	}

	return plaintext, nil
}

func (f *fakeSecrets) DeleteSecret(_ context.Context, secretRef string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}

	delete(f.values, secretRef)
	delete(f.owners, secretRef)
	f.deletedRefs = append(f.deletedRefs, secretRef)

	return nil
}

func (f *fakeSecrets) DeleteByOwner(_ context.Context, ownerType string, ownerID uuid.UUID) error {
	if f.deleteByOwnerErr != nil {
		return f.deleteByOwnerErr
	}

	owner := secretOwner{ownerType: ownerType, ownerID: ownerID}
	for ref, o := range f.owners {
		if o == owner {
			delete(f.values, ref)
			delete(f.owners, ref)
		}
	}

	f.deletedOwners = append(f.deletedOwners, owner)

	return nil
}

// seed registers an existing secret owned by the given destination and returns its reference.
func (f *fakeSecrets) seed(ownerID uuid.UUID, plaintext string) string {
	ref := secretutil.MakeSecretRef(uuid.New())
	f.values[ref] = plaintext
	f.owners[ref] = secretOwner{ownerType: ownerType, ownerID: ownerID}

	return ref
}

func (f *fakeSecrets) countForOwner(ownerID uuid.UUID) int {
	count := 0

	for _, o := range f.owners {
		if o.ownerType == ownerType && o.ownerID == ownerID {
			count++
		}
	}

	return count
}

type captureAudit struct {
	events []pipelineservice.AuditEvent
}

func (c *captureAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	c.events = append(c.events, event)
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

func newLogger() *logging.Logger {
	return &logging.Logger{Delegate: noopDelegate{}}
}

func ptr[T any](v T) *T {
	return &v
}

type connStorage struct {
	pipelineservice.ConnectionsStorage
	records []*pipelineservice.Connection
	findErr error
}

func (s *connStorage) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}

	var out []*pipelineservice.Connection

	for _, rec := range s.records {
		if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.WorkspaceID != eq.Value {
			continue
		}

		if eq, ok := f.DestinationID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.DestinationID != eq.Value {
			continue
		}

		cp := *rec
		out = append(out, &cp)
	}

	return out, nil
}
