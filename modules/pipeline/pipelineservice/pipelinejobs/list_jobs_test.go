package pipelinejobs

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

type listJobsStore struct {
	pipelineservice.Storage
	connections *listJobsConnections
	jobs        *listJobsJobs
	attempts    *listJobsAttempts
}

func (s *listJobsStore) Connections() pipelineservice.ConnectionsStorage { return s.connections }
func (s *listJobsStore) Jobs() pipelineservice.JobsStorage               { return s.jobs }
func (s *listJobsStore) JobAttempts() pipelineservice.JobAttemptsStorage { return s.attempts }

type listJobsConnections struct {
	pipelineservice.ConnectionsStorage
	records []*pipelineservice.Connection
}

func (m *listJobsConnections) matches(conn *pipelineservice.Connection, f *pipelineservice.ConnectionFilter) bool {
	if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.ID != eq.Value {
		return false
	}

	if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != eq.Value {
		return false
	}

	return true
}

func (m *listJobsConnections) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	for _, conn := range m.records {
		if m.matches(conn, f) {
			return conn, nil
		}
	}

	return nil, pipelineservice.ErrConnectionNotFound
}

func (m *listJobsConnections) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	var out []*pipelineservice.Connection

	for _, conn := range m.records {
		if m.matches(conn, f) {
			out = append(out, conn)
		}
	}

	return out, nil
}

type listJobsJobs struct {
	pipelineservice.JobsStorage
	records   []*pipelineservice.Job
	findCalls int
}

func (m *listJobsJobs) filtered(f *pipelineservice.JobFilter) []*pipelineservice.Job {
	var out []*pipelineservice.Job

	for _, job := range m.records {
		if in, ok := f.ConnectionID.(*filter.InFilter[uuid.UUID]); ok && !slices.Contains(in.Values, job.ConnectionID) {
			continue
		}

		if in, ok := f.Status.(*filter.InFilter[pipelineservice.JobStatus]); ok && !slices.Contains(in.Values, job.Status) {
			continue
		}

		out = append(out, job)
	}

	slices.SortFunc(out, func(a, b *pipelineservice.Job) int { return b.CreatedAt.Compare(a.CreatedAt) })

	return out
}

func (m *listJobsJobs) Count(_ context.Context, f *pipelineservice.JobFilter) (int, error) {
	return len(m.filtered(f)), nil
}

func (m *listJobsJobs) Find(_ context.Context, f *pipelineservice.JobFilter, opts ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Job, error) {
	m.findCalls++

	out := m.filtered(f)

	options := optionutil.ApplyOptions(&dbutil.SelectOptions{}, opts...)
	if p := options.Pagination; p != nil {
		start := min((p.Page-1)*p.PerPage, len(out))
		out = out[start:min(start+p.PerPage, len(out))]
	}

	return out, nil
}

type listJobsAttempts struct {
	pipelineservice.JobAttemptsStorage
	records []*pipelineservice.JobAttempt
}

func (m *listJobsAttempts) Find(_ context.Context, f *pipelineservice.JobAttemptFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.JobAttempt, error) {
	in := f.JobID.(*filter.InFilter[uuid.UUID])

	var out []*pipelineservice.JobAttempt

	for _, a := range m.records {
		if slices.Contains(in.Values, a.JobID) {
			out = append(out, a)
		}
	}

	return out, nil
}

type listJobsFixture struct {
	workspaceID uuid.UUID
	connA       uuid.UUID
	connB       uuid.UUID
	store       *listJobsStore
	useCase     *ListJobsWithAttempts
}

// newListJobsFixture creates two connections in one workspace and one in another.
// Jobs are created newest-first: jobs[0] is the most recent.
func newListJobsFixture(statuses ...pipelineservice.JobStatus) (*listJobsFixture, []*pipelineservice.Job) {
	f := &listJobsFixture{workspaceID: uuid.New(), connA: uuid.New(), connB: uuid.New()}
	otherConn := uuid.New()

	f.store = &listJobsStore{
		connections: &listJobsConnections{records: []*pipelineservice.Connection{
			{ID: f.connA, WorkspaceID: f.workspaceID},
			{ID: f.connB, WorkspaceID: f.workspaceID},
			{ID: otherConn, WorkspaceID: uuid.New()},
		}},
		jobs:     &listJobsJobs{},
		attempts: &listJobsAttempts{},
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jobs := make([]*pipelineservice.Job, len(statuses))

	for i, status := range statuses {
		conn := f.connA
		if i%2 == 1 {
			conn = f.connB
		}

		jobs[i] = &pipelineservice.Job{
			ID:           uuid.New(),
			ConnectionID: conn,
			Status:       status,
			CreatedAt:    base.Add(-time.Duration(i) * time.Minute),
		}
		f.store.attempts.records = append(f.store.attempts.records, &pipelineservice.JobAttempt{ID: uuid.New(), JobID: jobs[i].ID, AttemptNumber: 1})
	}

	f.store.jobs.records = append(slices.Clone(jobs), &pipelineservice.Job{
		ID: uuid.New(), ConnectionID: otherConn, Status: pipelineservice.JobStatusCompleted, CreatedAt: base,
	})

	f.useCase = NewListJobsWithAttempts(
		pipelineconnections.NewGetConnection(f.store),
		pipelineconnections.NewListConnections(f.store),
		NewListJobs(f.store),
		f.store,
	)

	return f, jobs
}

func jobIDsOf(items []*JobWithAttempts) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = item.Job.ID
	}

	return ids
}

func jobIDs(jobs []*pipelineservice.Job) []uuid.UUID {
	ids := make([]uuid.UUID, len(jobs))
	for i, job := range jobs {
		ids[i] = job.ID
	}

	return ids
}

func repeatStatus(status pipelineservice.JobStatus, n int) []pipelineservice.JobStatus {
	out := make([]pipelineservice.JobStatus, n)
	for i := range out {
		out[i] = status
	}

	return out
}

func TestListJobsWithAttempts_AllConnectionsInWorkspace(t *testing.T) {
	f, jobs := newListJobsFixture(repeatStatus(pipelineservice.JobStatusCompleted, 4)...)

	result, err := f.useCase.Execute(context.Background(), ListJobsWithAttemptsParams{
		WorkspaceID: f.workspaceID,
		Limit:       10,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result.Total)
	assert.Equal(t, jobIDs(jobs), jobIDsOf(result.Jobs))

	for _, item := range result.Jobs {
		require.Len(t, item.Attempts, 1)
		assert.Equal(t, item.Job.ID, item.Attempts[0].JobID)
	}
}

func TestListJobsWithAttempts_SingleConnection(t *testing.T) {
	f, jobs := newListJobsFixture(repeatStatus(pipelineservice.JobStatusCompleted, 4)...)

	result, err := f.useCase.Execute(context.Background(), ListJobsWithAttemptsParams{
		WorkspaceID:  f.workspaceID,
		ConnectionID: &f.connB,
		Limit:        10,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, []uuid.UUID{jobs[1].ID, jobs[3].ID}, jobIDsOf(result.Jobs))
}

func TestListJobsWithAttempts_RejectsForeignConnection(t *testing.T) {
	f, _ := newListJobsFixture(pipelineservice.JobStatusCompleted)
	foreign := f.store.connections.records[2].ID

	_, err := f.useCase.Execute(context.Background(), ListJobsWithAttemptsParams{
		WorkspaceID:  f.workspaceID,
		ConnectionID: &foreign,
		Limit:        10,
	})
	require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
}

func TestListJobsWithAttempts_StatusFilter(t *testing.T) {
	f, jobs := newListJobsFixture(
		pipelineservice.JobStatusRunning,
		pipelineservice.JobStatusFailed,
		pipelineservice.JobStatusStarting,
		pipelineservice.JobStatusCompleted,
	)

	result, err := f.useCase.Execute(context.Background(), ListJobsWithAttemptsParams{
		WorkspaceID: f.workspaceID,
		Statuses:    []pipelineservice.JobStatus{pipelineservice.JobStatusRunning, pipelineservice.JobStatusStarting},
		Limit:       10,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, []uuid.UUID{jobs[0].ID, jobs[2].ID}, jobIDsOf(result.Jobs))
}

func TestListJobsWithAttempts_Pagination(t *testing.T) {
	f, jobs := newListJobsFixture(repeatStatus(pipelineservice.JobStatusCompleted, 7)...)

	cases := []struct {
		name   string
		limit  int
		offset int
		want   []*pipelineservice.Job
	}{
		{name: "first page", limit: 3, offset: 0, want: jobs[0:3]},
		{name: "aligned page", limit: 3, offset: 3, want: jobs[3:6]},
		{name: "partial last page", limit: 3, offset: 6, want: jobs[6:7]},
		{name: "unaligned offset", limit: 3, offset: 2, want: jobs[2:5]},
		{name: "unaligned offset near end", limit: 3, offset: 5, want: jobs[5:7]},
		{name: "offset past end", limit: 3, offset: 7, want: nil},
		{name: "negative offset", limit: 3, offset: -4, want: jobs[0:3]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := f.useCase.Execute(context.Background(), ListJobsWithAttemptsParams{
				WorkspaceID: f.workspaceID,
				Limit:       tc.limit,
				Offset:      tc.offset,
			})
			require.NoError(t, err)
			assert.Equal(t, 7, result.Total)
			assert.Equal(t, jobIDs(tc.want), jobIDsOf(result.Jobs))
		})
	}
}

func TestListJobsWithAttempts_EmptyWorkspaceSkipsJobQuery(t *testing.T) {
	f, _ := newListJobsFixture(pipelineservice.JobStatusCompleted)

	result, err := f.useCase.Execute(context.Background(), ListJobsWithAttemptsParams{
		WorkspaceID: uuid.New(),
		Limit:       10,
	})
	require.NoError(t, err)
	assert.Zero(t, result.Total)
	assert.Empty(t, result.Jobs)
	assert.Zero(t, f.store.jobs.findCalls)
}
