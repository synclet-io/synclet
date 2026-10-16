package pipelinetasks_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

var errBoom = errors.New("boom")

type taskStore struct {
	pipelineservice.Storage
	tasks        *taskStorage
	connectors   *connectorStorage
	sources      *sourceStorage
	destinations *destinationStorage
	catalogs     *catalogStorage
}

func newTaskStore(records ...*pipelineservice.ConnectorTask) *taskStore {
	m := make(map[uuid.UUID]*pipelineservice.ConnectorTask, len(records))
	for _, r := range records {
		m[r.ID] = r
	}

	return &taskStore{
		tasks:        &taskStorage{records: m},
		connectors:   &connectorStorage{records: map[uuid.UUID]*pipelineservice.ManagedConnector{}},
		sources:      &sourceStorage{records: map[uuid.UUID]*pipelineservice.Source{}},
		destinations: &destinationStorage{records: map[uuid.UUID]*pipelineservice.Destination{}},
		catalogs:     &catalogStorage{},
	}
}

func (s *taskStore) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return s.tasks
}

func (s *taskStore) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return s.connectors
}

func (s *taskStore) Sources() pipelineservice.SourcesStorage {
	return s.sources
}

func (s *taskStore) Destinations() pipelineservice.DestinationsStorage {
	return s.destinations
}

func (s *taskStore) CatalogDiscoveries() pipelineservice.CatalogDiscoveriesStorage {
	return s.catalogs
}

func (s *taskStore) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

func matches[E comparable](f filter.Filter[E], v E) bool {
	switch f := f.(type) {
	case nil:
		return true
	case *filter.EqualsFilter[E]:
		return v == f.Value
	case *filter.InFilter[E]:
		for _, want := range f.Values {
			if v == want {
				return true
			}
		}

		return false
	default:
		panic("unsupported filter")
	}
}

func timeMatches(f filter.Filter[time.Time], v time.Time) bool {
	switch f := f.(type) {
	case nil:
		return true
	case *filter.LessFilter[time.Time]:
		return v.Before(f.Value)
	default:
		panic("unsupported time filter")
	}
}

func optTimeMatches(f filter.Filter[*time.Time], v *time.Time) bool {
	switch f := f.(type) {
	case nil:
		return true
	case *filter.LessFilter[*time.Time]:
		return v != nil && v.Before(*f.Value)
	default:
		panic("unsupported time filter")
	}
}

// taskStorage is safe for concurrent use; ClaimPendingTask hands each pending
// task to exactly one caller, like the SKIP LOCKED query it stands in for.
type taskStorage struct {
	pipelineservice.ConnectorTasksStorage
	mu        sync.Mutex
	records   map[uuid.UUID]*pipelineservice.ConnectorTask
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	claimErr  error
}

func (s *taskStorage) match(f *pipelineservice.ConnectorTaskFilter, rec *pipelineservice.ConnectorTask) bool {
	return matches(f.ID, rec.ID) &&
		matches(f.WorkspaceID, rec.WorkspaceID) &&
		matches(f.TaskType, rec.TaskType) &&
		matches(f.Status, rec.Status) &&
		timeMatches(f.CreatedAt, rec.CreatedAt) &&
		timeMatches(f.UpdatedAt, rec.UpdatedAt) &&
		optTimeMatches(f.CompletedAt, rec.CompletedAt)
}

func (s *taskStorage) get(id uuid.UUID) *pipelineservice.ConnectorTask {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.records[id]
	if !ok {
		return nil
	}

	cp := *rec

	return &cp
}

func (s *taskStorage) set(id uuid.UUID, mutate func(*pipelineservice.ConnectorTask)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mutate(s.records[id])
}

func (s *taskStorage) First(_ context.Context, f *pipelineservice.ConnectorTaskFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConnectorTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rec := range s.records {
		if !s.match(f, rec) {
			continue
		}

		cp := *rec

		return &cp, nil
	}

	return nil, pipelineservice.ErrConnectorTaskNotFound
}

func (s *taskStorage) Find(_ context.Context, f *pipelineservice.ConnectorTaskFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ConnectorTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findErr != nil {
		return nil, s.findErr
	}

	var out []*pipelineservice.ConnectorTask

	for _, rec := range s.records {
		if s.match(f, rec) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *taskStorage) Create(_ context.Context, task *pipelineservice.ConnectorTask) (*pipelineservice.ConnectorTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.createErr != nil {
		return nil, s.createErr
	}

	cp := *task
	s.records[task.ID] = &cp

	return task, nil
}

func (s *taskStorage) Update(_ context.Context, task *pipelineservice.ConnectorTask) (*pipelineservice.ConnectorTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.updateErr != nil {
		return nil, s.updateErr
	}

	cp := *task
	s.records[task.ID] = &cp

	return task, nil
}

func (s *taskStorage) Delete(_ context.Context, f *pipelineservice.ConnectorTaskFilter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.deleteErr != nil {
		return s.deleteErr
	}

	for id, rec := range s.records {
		if s.match(f, rec) {
			delete(s.records, id)
		}
	}

	return nil
}

func (s *taskStorage) ClaimPendingTask(_ context.Context, workerID string) (*pipelineservice.ConnectorTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.claimErr != nil {
		return nil, s.claimErr
	}

	var pending []*pipelineservice.ConnectorTask

	for _, rec := range s.records {
		if rec.Status == pipelineservice.ConnectorTaskStatusPending {
			pending = append(pending, rec)
		}
	}

	if len(pending) == 0 {
		return nil, nil
	}

	sort.Slice(pending, func(i, j int) bool { return pending[i].CreatedAt.Before(pending[j].CreatedAt) })

	claimed := pending[0]
	claimed.Status = pipelineservice.ConnectorTaskStatusRunning
	claimed.WorkerID = &workerID
	claimed.UpdatedAt = time.Now()

	cp := *claimed

	return &cp, nil
}

type connectorStorage struct {
	pipelineservice.ManagedConnectorsStorage
	records map[uuid.UUID]*pipelineservice.ManagedConnector
}

func (s *connectorStorage) First(_ context.Context, f *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	for _, rec := range s.records {
		if matches(f.ID, rec.ID) && matches(f.WorkspaceID, rec.WorkspaceID) {
			cp := *rec

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrManagedConnectorNotFound
}

type sourceStorage struct {
	pipelineservice.SourcesStorage
	records map[uuid.UUID]*pipelineservice.Source
}

func (s *sourceStorage) First(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	for _, rec := range s.records {
		if matches(f.ID, rec.ID) && matches(f.WorkspaceID, rec.WorkspaceID) {
			cp := *rec

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrSourceNotFound
}

type destinationStorage struct {
	pipelineservice.DestinationsStorage
	records map[uuid.UUID]*pipelineservice.Destination
}

func (s *destinationStorage) First(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Destination, error) {
	for _, rec := range s.records {
		if matches(f.ID, rec.ID) && matches(f.WorkspaceID, rec.WorkspaceID) {
			cp := *rec

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrDestinationNotFound
}

type catalogStorage struct {
	pipelineservice.CatalogDiscoveriesStorage
	records   []*pipelineservice.CatalogDiscovery
	createErr error
}

func (s *catalogStorage) First(_ context.Context, f *pipelineservice.CatalogDiscoveryFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.CatalogDiscovery, error) {
	var latest *pipelineservice.CatalogDiscovery

	for _, rec := range s.records {
		if matches(f.SourceID, rec.SourceID) && (latest == nil || rec.Version > latest.Version) {
			latest = rec
		}
	}

	if latest == nil {
		return nil, pipelineservice.ErrCatalogDiscoveryNotFound
	}

	cp := *latest

	return &cp, nil
}

func (s *catalogStorage) Create(_ context.Context, rec *pipelineservice.CatalogDiscovery) (*pipelineservice.CatalogDiscovery, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}

	s.records = append(s.records, rec)

	return rec, nil
}

type fakeSecrets struct {
	pipelineservice.SecretsProvider
	mu        sync.Mutex
	values    map[string]string
	deleted   []uuid.UUID
	deleteErr error
}

func newFakeSecrets() *fakeSecrets {
	return &fakeSecrets{values: map[string]string{}}
}

func (s *fakeSecrets) StoreSecret(_ context.Context, _ string, _ uuid.UUID, plaintext string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref := secretutil.MakeSecretRef(uuid.New())
	s.values[ref] = plaintext

	return ref, nil
}

func (s *fakeSecrets) RetrieveSecret(_ context.Context, ref string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.values[ref]
	if !ok {
		return "", errBoom
	}

	return v, nil
}

func (s *fakeSecrets) DeleteByOwner(_ context.Context, ownerType string, ownerID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.deleteErr != nil {
		return s.deleteErr
	}

	if ownerType == "connector_task" {
		s.deleted = append(s.deleted, ownerID)
	}

	return nil
}

func (s *fakeSecrets) deletedOwners() []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]uuid.UUID(nil), s.deleted...)
}

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (a *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	a.events = append(a.events, event)
}

type noopLogDelegate struct{}

func (noopLogDelegate) Info(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopLogDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopLogDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopLogDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopLogDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopLogDelegate) Named(string) logging.Delegate                      { return d }
func (d noopLogDelegate) SkipCallers(int) logging.Delegate                   { return d }

func testLogger() *logging.Logger {
	return &logging.Logger{Delegate: noopLogDelegate{}}
}

func strPtr(s string) *string {
	return &s
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
