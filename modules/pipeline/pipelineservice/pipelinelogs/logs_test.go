package pipelinelogs_test

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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
)

type logStore struct {
	pipelineservice.Storage
	jobs        *logJobsStorage
	connections *logConnectionsStorage
	jobLogs     *logJobLogsStorage
}

func (s *logStore) Jobs() pipelineservice.JobsStorage               { return s.jobs }
func (s *logStore) Connections() pipelineservice.ConnectionsStorage { return s.connections }
func (s *logStore) JobLogs() pipelineservice.JobLogsStorage         { return s.jobLogs }

type logJobsStorage struct {
	pipelineservice.JobsStorage
	records map[uuid.UUID]*pipelineservice.Job
}

func (m *logJobsStorage) First(_ context.Context, f *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Job, error) {
	id := f.ID.(*filter.EqualsFilter[uuid.UUID]).Value
	rec, ok := m.records[id]

	if !ok {
		return nil, pipelineservice.ErrJobNotFound
	}

	cp := *rec

	return &cp, nil
}

type logConnectionsStorage struct {
	pipelineservice.ConnectionsStorage
	records map[uuid.UUID]*pipelineservice.Connection
}

func (m *logConnectionsStorage) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	for _, conn := range m.records {
		if f.ID != nil {
			if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.ID != eq.Value {
				continue
			}
		}

		if f.WorkspaceID != nil {
			if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != eq.Value {
				continue
			}
		}

		cp := *conn

		return &cp, nil
	}

	return nil, pipelineservice.ErrConnectionNotFound
}

type logJobLogsStorage struct {
	pipelineservice.JobLogsStorage
	logs map[uuid.UUID][]pipelineservice.JobLog
	next int64
}

func (m *logJobLogsStorage) AppendLog(_ context.Context, jobID uuid.UUID, line string) error {
	m.next++
	m.logs[jobID] = append(m.logs[jobID], pipelineservice.JobLog{ID: m.next, JobID: jobID, LogLine: line})

	return nil
}

func (m *logJobLogsStorage) BatchAppendLogs(_ context.Context, jobID uuid.UUID, lines []string) error {
	for _, l := range lines {
		m.next++
		m.logs[jobID] = append(m.logs[jobID], pipelineservice.JobLog{ID: m.next, JobID: jobID, LogLine: l})
	}

	return nil
}

func (m *logJobLogsStorage) GetLogs(_ context.Context, jobID uuid.UUID, afterID int64, limit int) ([]pipelineservice.JobLog, error) {
	out := make([]pipelineservice.JobLog, 0)

	for _, log := range m.logs[jobID] {
		if log.ID <= afterID {
			continue
		}

		out = append(out, log)
		if limit > 0 && len(out) >= limit {
			break
		}
	}

	return out, nil
}

func newLogStore(workspaceID, jobID, connID uuid.UUID) *logStore {
	return &logStore{
		jobs: &logJobsStorage{records: map[uuid.UUID]*pipelineservice.Job{
			jobID: {ID: jobID, ConnectionID: connID},
		}},
		connections: &logConnectionsStorage{records: map[uuid.UUID]*pipelineservice.Connection{
			connID: {ID: connID, WorkspaceID: workspaceID},
		}},
		jobLogs: &logJobLogsStorage{logs: map[uuid.UUID][]pipelineservice.JobLog{}},
	}
}

func TestAppendAndGetJobLog(t *testing.T) {
	workspaceID := uuid.New()
	jobID := uuid.New()
	connID := uuid.New()
	store := newLogStore(workspaceID, jobID, connID)

	appendUC := pipelinelogs.NewAppendJobLog(store)
	require.NoError(t, appendUC.Execute(context.Background(), pipelinelogs.AppendJobLogParams{JobID: jobID, LogLine: "line-1"}))
	require.NoError(t, appendUC.Execute(context.Background(), pipelinelogs.AppendJobLogParams{JobID: jobID, LogLine: "line-2"}))

	getUC := pipelinelogs.NewGetJobLog(store)
	result, err := getUC.Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: workspaceID,
		JobID:       jobID,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"line-1", "line-2"}, result.Lines)
	assert.Equal(t, int64(2), result.LastID)
	assert.False(t, result.HasMore)
}

func TestBatchAppendJobLogs(t *testing.T) {
	workspaceID := uuid.New()
	jobID := uuid.New()
	connID := uuid.New()
	store := newLogStore(workspaceID, jobID, connID)

	batch := pipelinelogs.NewBatchAppendJobLogs(store)
	require.NoError(t, batch.Execute(context.Background(), pipelinelogs.BatchAppendJobLogsParams{
		JobID:    jobID,
		LogLines: []string{"a", "b", "c"},
	}))

	getUC := pipelinelogs.NewGetJobLog(store)
	result, err := getUC.Execute(context.Background(), pipelinelogs.GetJobLogParams{WorkspaceID: workspaceID, JobID: jobID})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result.Lines)
}

func TestGetJobLog_PaginationWithCursor(t *testing.T) {
	workspaceID := uuid.New()
	jobID := uuid.New()
	connID := uuid.New()
	store := newLogStore(workspaceID, jobID, connID)

	batch := pipelinelogs.NewBatchAppendJobLogs(store)
	require.NoError(t, batch.Execute(context.Background(), pipelinelogs.BatchAppendJobLogsParams{
		JobID:    jobID,
		LogLines: []string{"a", "b", "c", "d", "e"},
	}))

	getUC := pipelinelogs.NewGetJobLog(store)
	first, err := getUC.Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: workspaceID,
		JobID:       jobID,
		Limit:       2,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, first.Lines)
	assert.Equal(t, int64(2), first.LastID)
	assert.True(t, first.HasMore)

	second, err := getUC.Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: workspaceID,
		JobID:       jobID,
		AfterID:     first.LastID,
		Limit:       2,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "d"}, second.Lines)
	assert.True(t, second.HasMore)

	third, err := getUC.Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: workspaceID,
		JobID:       jobID,
		AfterID:     second.LastID,
		Limit:       2,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"e"}, third.Lines)
	assert.False(t, third.HasMore)
}

func TestGetJobLog_RefusesCrossWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	jobID := uuid.New()
	connID := uuid.New()
	store := newLogStore(workspaceID, jobID, connID)

	require.NoError(t, pipelinelogs.NewAppendJobLog(store).Execute(context.Background(), pipelinelogs.AppendJobLogParams{JobID: jobID, LogLine: "hi"}))

	getUC := pipelinelogs.NewGetJobLog(store)
	_, err := getUC.Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: uuid.New(), // different workspace
		JobID:       jobID,
	})
	require.Error(t, err)
}
