package pipelinegroups_test

import (
	"context"
	"slices"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func matches[E comparable](f filter.Filter[E], value E) bool {
	switch typed := f.(type) {
	case nil:
		return true
	case *filter.EqualsFilter[E]:
		return typed.Value == value
	case *filter.InFilter[E]:
		return slices.Contains(typed.Values, value)
	default:
		panic("unsupported filter")
	}
}

type store struct {
	pipelineservice.Storage
	sources      *sourceStorage
	destinations *destinationStorage
	connections  *connectionStorage
	jobs         *jobStorage
}

func newStore() *store {
	return &store{
		sources:      &sourceStorage{records: map[uuid.UUID]*pipelineservice.Source{}},
		destinations: &destinationStorage{records: map[uuid.UUID]*pipelineservice.Destination{}},
		connections:  &connectionStorage{},
		jobs:         &jobStorage{},
	}
}

func (s *store) Sources() pipelineservice.SourcesStorage           { return s.sources }
func (s *store) Destinations() pipelineservice.DestinationsStorage { return s.destinations }
func (s *store) Connections() pipelineservice.ConnectionsStorage   { return s.connections }
func (s *store) Jobs() pipelineservice.JobsStorage                 { return s.jobs }

// ExecuteInTransaction runs cb on a copy of the sources and destinations and
// keeps the copy only when cb succeeds.
func (s *store) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	tx := &store{
		sources:      &sourceStorage{records: cloneMap(s.sources.records)},
		destinations: &destinationStorage{records: cloneMap(s.destinations.records), updateErr: s.destinations.updateErr},
		connections:  s.connections,
		jobs:         s.jobs,
	}

	if err := cb(ctx, tx); err != nil {
		return err
	}

	s.sources.records = tx.sources.records
	s.destinations.records = tx.destinations.records

	return nil
}

func cloneMap[T any](in map[uuid.UUID]*T) map[uuid.UUID]*T {
	out := make(map[uuid.UUID]*T, len(in))

	for id, rec := range in {
		cp := *rec
		out[id] = &cp
	}

	return out
}

func (s *store) addSource(ws uuid.UUID, group string) *pipelineservice.Source {
	src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, Name: "src-" + group, ExecutorGroup: group}
	s.sources.records[src.ID] = src

	return src
}

func (s *store) addDestination(ws uuid.UUID, group string) *pipelineservice.Destination {
	dest := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, Name: "dest-" + group, ExecutorGroup: group}
	s.destinations.records[dest.ID] = dest

	return dest
}

func (s *store) addConnection(src *pipelineservice.Source, dest *pipelineservice.Destination) *pipelineservice.Connection {
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: src.WorkspaceID, Name: "c", SourceID: src.ID, DestinationID: dest.ID}
	s.connections.records = append(s.connections.records, conn)

	return conn
}

func (s *store) addJob(conn *pipelineservice.Connection, group string, status pipelineservice.JobStatus) {
	s.jobs.records = append(s.jobs.records, &pipelineservice.Job{ID: uuid.New(), ConnectionID: conn.ID, ExecutorGroup: group, Status: status})
}

type sourceStorage struct {
	pipelineservice.SourcesStorage
	records map[uuid.UUID]*pipelineservice.Source
}

func (s *sourceStorage) Find(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	var out []*pipelineservice.Source

	for _, rec := range s.records {
		if matches(f.ID, rec.ID) && matches(f.WorkspaceID, rec.WorkspaceID) && matches(f.ExecutorGroup, rec.ExecutorGroup) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *sourceStorage) Update(_ context.Context, model *pipelineservice.Source) (*pipelineservice.Source, error) {
	cp := *model
	s.records[model.ID] = &cp

	return model, nil
}

type destinationStorage struct {
	pipelineservice.DestinationsStorage
	records   map[uuid.UUID]*pipelineservice.Destination
	updateErr error
}

func (s *destinationStorage) Find(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	var out []*pipelineservice.Destination

	for _, rec := range s.records {
		if matches(f.ID, rec.ID) && matches(f.WorkspaceID, rec.WorkspaceID) && matches(f.ExecutorGroup, rec.ExecutorGroup) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (s *destinationStorage) Update(_ context.Context, model *pipelineservice.Destination) (*pipelineservice.Destination, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}

	cp := *model
	s.records[model.ID] = &cp

	return model, nil
}

type connectionStorage struct {
	pipelineservice.ConnectionsStorage
	records []*pipelineservice.Connection
}

func (s *connectionStorage) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	var out []*pipelineservice.Connection

	for _, rec := range s.records {
		if matches(f.WorkspaceID, rec.WorkspaceID) && matches(f.SourceID, rec.SourceID) && matches(f.DestinationID, rec.DestinationID) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

type jobStorage struct {
	pipelineservice.JobsStorage
	records []*pipelineservice.Job
}

func (s *jobStorage) Find(_ context.Context, f *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Job, error) {
	var out []*pipelineservice.Job

	for _, rec := range s.records {
		if matches(f.ConnectionID, rec.ConnectionID) && matches(f.ExecutorGroup, rec.ExecutorGroup) && matches(f.Status, rec.Status) {
			cp := *rec
			out = append(out, &cp)
		}
	}

	return out, nil
}

type captureAudit struct {
	events []pipelineservice.AuditEvent
}

func (c *captureAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	c.events = append(c.events, event)
}
