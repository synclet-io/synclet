// Package pipelinefake is an in-memory pipelineservice.Storage for use-case
// tests. It holds connections, jobs, workflows and their runs; every other
// storage panics when used. Transactions run the callback on the same store
// without rollback.
package pipelinefake

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// Storage is the in-memory store.
type Storage struct {
	pipelineservice.Storage

	ConnectionsTable              *Table[pipelineservice.Connection, pipelineservice.ConnectionFilter]
	JobsTable                     *Table[pipelineservice.Job, pipelineservice.JobFilter]
	WorkflowsTable                *Table[pipelineservice.Workflow, pipelineservice.WorkflowFilter]
	WorkflowStepsTable            *Table[pipelineservice.WorkflowStep, pipelineservice.WorkflowStepFilter]
	WorkflowStepDependenciesTable *Table[pipelineservice.WorkflowStepDependency, pipelineservice.WorkflowStepDependencyFilter]
	WorkflowRunsTable             *Table[pipelineservice.WorkflowRun, pipelineservice.WorkflowRunFilter]
	WorkflowStepRunsTable         *Table[pipelineservice.WorkflowStepRun, pipelineservice.WorkflowStepRunFilter]

	Events []*pipelineservice.PipelineEvent
	Locks  []string
}

// New returns an empty store.
func New() *Storage {
	return &Storage{
		ConnectionsTable: newTable(pipelineservice.ErrConnectionNotFound,
			func(e *pipelineservice.Connection) uuid.UUID { return e.ID }, matchConnection),
		JobsTable: newTable(pipelineservice.ErrJobNotFound,
			func(e *pipelineservice.Job) uuid.UUID { return e.ID }, matchJob),
		WorkflowsTable: newTable(pipelineservice.ErrWorkflowNotFound,
			func(e *pipelineservice.Workflow) uuid.UUID { return e.ID }, matchWorkflow),
		WorkflowStepsTable: newTable(pipelineservice.ErrWorkflowStepNotFound,
			func(e *pipelineservice.WorkflowStep) uuid.UUID { return e.ID }, matchWorkflowStep),
		WorkflowStepDependenciesTable: newTable(pipelineservice.ErrWorkflowStepDependencyNotFound,
			func(e *pipelineservice.WorkflowStepDependency) uuid.UUID { return e.ID }, matchWorkflowStepDependency),
		WorkflowRunsTable: newTable(pipelineservice.ErrWorkflowRunNotFound,
			func(e *pipelineservice.WorkflowRun) uuid.UUID { return e.ID }, matchWorkflowRun),
		WorkflowStepRunsTable: newTable(pipelineservice.ErrWorkflowStepRunNotFound,
			func(e *pipelineservice.WorkflowStepRun) uuid.UUID { return e.ID }, matchWorkflowStepRun),
	}
}

func (s *Storage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

func (s *Storage) WithAdvisoryLock(_ context.Context, scope string, _ int64) error {
	s.Locks = append(s.Locks, scope)

	return nil
}

func (s *Storage) Connections() pipelineservice.ConnectionsStorage {
	return &connections{Table: s.ConnectionsTable}
}

func (s *Storage) Jobs() pipelineservice.JobsStorage { return &jobs{Table: s.JobsTable} }

func (s *Storage) Workflows() pipelineservice.WorkflowsStorage { return s.WorkflowsTable }

func (s *Storage) WorkflowSteps() pipelineservice.WorkflowStepsStorage { return s.WorkflowStepsTable }

func (s *Storage) WorkflowStepDependencies() pipelineservice.WorkflowStepDependenciesStorage {
	return s.WorkflowStepDependenciesTable
}

func (s *Storage) WorkflowRuns() pipelineservice.WorkflowRunsStorage { return s.WorkflowRunsTable }

func (s *Storage) WorkflowStepRuns() pipelineservice.WorkflowStepRunsStorage {
	return s.WorkflowStepRunsTable
}

func (s *Storage) PipelineEvents() pipelineservice.PipelineEventsOutbox { return &outbox{store: s} }

type outbox struct{ store *Storage }

func (o *outbox) Send(_ context.Context, event *pipelineservice.PipelineEvent) error {
	o.store.Events = append(o.store.Events, event)

	return nil
}

type unimplementedConnections struct {
	pipelineservice.ConnectionsStorage
}

type connections struct {
	unimplementedConnections
	*Table[pipelineservice.Connection, pipelineservice.ConnectionFilter]
}

func (c *connections) FindDueConnections(context.Context, int) ([]pipelineservice.DueConnection, error) {
	return nil, nil
}

type unimplementedJobs struct{ pipelineservice.JobsStorage }

type jobs struct {
	unimplementedJobs
	*Table[pipelineservice.Job, pipelineservice.JobFilter]
}

func (j *jobs) CountActiveJobs(context.Context) (int, error) {
	count := 0

	for _, job := range j.Rows {
		if isActiveJob(job.Status) {
			count++
		}
	}

	return count, nil
}

func isActiveJob(status pipelineservice.JobStatus) bool {
	return status == pipelineservice.JobStatusScheduled ||
		status == pipelineservice.JobStatusStarting ||
		status == pipelineservice.JobStatusRunning
}

// Table is one in-memory table. Reads and writes copy rows, as a database does.
type Table[E any, F any] struct {
	Rows     []*E
	notFound error
	idOf     func(*E) uuid.UUID
	match    func(*F, *E) bool
}

func newTable[E any, F any](notFound error, idOf func(*E) uuid.UUID, match func(*F, *E) bool) *Table[E, F] {
	return &Table[E, F]{notFound: notFound, idOf: idOf, match: match}
}

// Insert adds rows directly, for test setup.
func (t *Table[E, F]) Insert(rows ...*E) {
	for _, row := range rows {
		t.Rows = append(t.Rows, clone(row))
	}
}

// Get returns a copy of the row with the given id, or nil.
func (t *Table[E, F]) Get(id uuid.UUID) *E {
	for _, row := range t.Rows {
		if t.idOf(row) == id {
			return clone(row)
		}
	}

	return nil
}

// All returns copies of every row.
func (t *Table[E, F]) All() []*E {
	result := make([]*E, 0, len(t.Rows))
	for _, row := range t.Rows {
		result = append(result, clone(row))
	}

	return result
}

func (t *Table[E, F]) Create(_ context.Context, model *E) (*E, error) {
	if t.Get(t.idOf(model)) != nil {
		return nil, fmt.Errorf("duplicate id %s", t.idOf(model))
	}

	t.Rows = append(t.Rows, clone(model))

	return clone(model), nil
}

func (t *Table[E, F]) BatchCreate(ctx context.Context, models []*E) ([]*E, error) {
	result := make([]*E, 0, len(models))

	for _, model := range models {
		created, err := t.Create(ctx, model)
		if err != nil {
			return nil, err
		}

		result = append(result, created)
	}

	return result, nil
}

func (t *Table[E, F]) Count(ctx context.Context, f *F) (int, error) {
	found, err := t.Find(ctx, f)

	return len(found), err
}

func (t *Table[E, F]) Update(_ context.Context, model *E) (*E, error) {
	for i, row := range t.Rows {
		if t.idOf(row) == t.idOf(model) {
			t.Rows[i] = clone(model)

			return clone(model), nil
		}
	}

	return clone(model), nil
}

func (t *Table[E, F]) Save(ctx context.Context, model *E) (*E, error) {
	if t.Get(t.idOf(model)) == nil {
		return t.Create(ctx, model)
	}

	return t.Update(ctx, model)
}

func (t *Table[E, F]) First(ctx context.Context, f *F, _ ...optionutil.Option[dbutil.SelectOptions]) (*E, error) {
	found, _ := t.Find(ctx, f)
	if len(found) == 0 {
		return nil, t.notFound
	}

	return found[0], nil
}

func (t *Table[E, F]) FirstOrCreate(ctx context.Context, f *F, model *E, _ ...optionutil.Option[dbutil.SelectOptions]) (*E, error) {
	found, _ := t.Find(ctx, f)
	if len(found) > 0 {
		return found[0], nil
	}

	return t.Create(ctx, model)
}

func (t *Table[E, F]) Find(_ context.Context, f *F, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*E, error) {
	var result []*E

	for _, row := range t.Rows {
		if f == nil || t.match(f, row) {
			result = append(result, clone(row))
		}
	}

	return result, nil
}

func (t *Table[E, F]) Delete(_ context.Context, f *F) error {
	kept := t.Rows[:0]

	for _, row := range t.Rows {
		if f == nil || !t.match(f, row) {
			kept = append(kept, row)
		}
	}

	t.Rows = kept

	return nil
}

func (t *Table[E, F]) WithAdvisoryLock(context.Context, int64) error { return nil }

func clone[E any](row *E) *E {
	c := *row

	return &c
}

func value[E comparable](cond filter.Filter[E], got E) bool {
	switch typed := cond.(type) {
	case nil:
		return true
	case *filter.EqualsFilter[E]:
		return typed.Value == got
	case *filter.NotEqualsFilter[E]:
		return typed.Value != got
	case *filter.InFilter[E]:
		return slices.Contains(typed.Values, got)
	case *filter.NotInFilter[E]:
		return !slices.Contains(typed.Values, got)
	default:
		panic(fmt.Sprintf("pipelinefake: unsupported filter %T", cond))
	}
}

func pointer[E comparable](cond filter.Filter[*E], got *E) bool {
	switch typed := cond.(type) {
	case nil:
		return true
	case *filter.IsNullFilter[*E]:
		return got == nil
	case *filter.IsNotNullFilter[*E]:
		return got != nil
	case *filter.EqualsFilter[*E]:
		return got != nil && typed.Value != nil && *typed.Value == *got
	case *filter.InFilter[*E]:
		return got != nil && slices.ContainsFunc(typed.Values, func(candidate *E) bool { return candidate != nil && *candidate == *got })
	default:
		panic(fmt.Sprintf("pipelinefake: unsupported filter %T", cond))
	}
}

func timestamp(cond filter.Filter[*time.Time], got *time.Time) bool {
	switch typed := cond.(type) {
	case nil:
		return true
	case *filter.IsNullFilter[*time.Time]:
		return got == nil
	case *filter.IsNotNullFilter[*time.Time]:
		return got != nil
	case *filter.LessOrEqualsFilter[*time.Time]:
		return got != nil && !got.After(*typed.Value)
	case *filter.LessFilter[*time.Time]:
		return got != nil && got.Before(*typed.Value)
	default:
		panic(fmt.Sprintf("pipelinefake: unsupported filter %T", cond))
	}
}

func combine[F any, E any](row *E, or, and []*F, fields bool, match func(*F, *E) bool) bool {
	if !fields {
		return false
	}

	if len(or) > 0 && !slices.ContainsFunc(or, func(sub *F) bool { return match(sub, row) }) {
		return false
	}

	for _, sub := range and {
		if !match(sub, row) {
			return false
		}
	}

	return true
}

func matchConnection(cond *pipelineservice.ConnectionFilter, row *pipelineservice.Connection) bool {
	if cond.Tags != nil {
		panic("pipelinefake: tag filters are not supported")
	}

	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.WorkspaceID, row.WorkspaceID) && value(cond.Name, row.Name) &&
			value(cond.Status, row.Status) && value(cond.SourceID, row.SourceID) && value(cond.DestinationID, row.DestinationID),
		matchConnection)
}

func matchJob(cond *pipelineservice.JobFilter, row *pipelineservice.Job) bool {
	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.ConnectionID, row.ConnectionID) && value(cond.Status, row.Status) &&
			value(cond.JobType, row.JobType) && timestamp(cond.StartedAt, row.StartedAt) &&
			timestamp(cond.HeartbeatAt, row.HeartbeatAt) && pointer(cond.WorkflowStepRunID, row.WorkflowStepRunID),
		matchJob)
}

func matchWorkflow(cond *pipelineservice.WorkflowFilter, row *pipelineservice.Workflow) bool {
	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.WorkspaceID, row.WorkspaceID) && value(cond.Name, row.Name) &&
			timestamp(cond.NextScheduledAt, row.NextScheduledAt),
		matchWorkflow)
}

func matchWorkflowStep(cond *pipelineservice.WorkflowStepFilter, row *pipelineservice.WorkflowStep) bool {
	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.WorkflowID, row.WorkflowID) && value(cond.ConnectionID, row.ConnectionID),
		matchWorkflowStep)
}

func matchWorkflowStepDependency(cond *pipelineservice.WorkflowStepDependencyFilter, row *pipelineservice.WorkflowStepDependency) bool {
	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.WorkflowID, row.WorkflowID) && value(cond.StepID, row.StepID) &&
			value(cond.UpstreamStepID, row.UpstreamStepID),
		matchWorkflowStepDependency)
}

func matchWorkflowRun(cond *pipelineservice.WorkflowRunFilter, row *pipelineservice.WorkflowRun) bool {
	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.WorkflowID, row.WorkflowID) && value(cond.Status, row.Status) &&
			value(cond.Trigger, row.Trigger),
		matchWorkflowRun)
}

func matchWorkflowStepRun(cond *pipelineservice.WorkflowStepRunFilter, row *pipelineservice.WorkflowStepRun) bool {
	return combine(row, cond.Or, cond.And,
		value(cond.ID, row.ID) && value(cond.RunID, row.RunID) && value(cond.StepID, row.StepID) &&
			value(cond.ConnectionID, row.ConnectionID) && value(cond.Status, row.Status) && pointer(cond.JobID, row.JobID),
		matchWorkflowStepRun)
}
