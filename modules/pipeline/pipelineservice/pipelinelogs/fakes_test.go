package pipelinelogs_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/logarchive"
)

type segmentKey struct {
	jobID   uuid.UUID
	attempt int
}

// logStore is an in-memory pipelineservice.Storage covering jobs,
// connections, job logs and job log segments.
type logStore struct {
	pipelineservice.Storage

	jobs        map[uuid.UUID]*pipelineservice.Job
	connections map[uuid.UUID]*pipelineservice.Connection
	logs        []pipelineservice.JobLog
	segments    map[segmentKey]*pipelineservice.JobLogSegment
	nextID      int64

	markArchivedErr error
	beforeMark      func()
}

func newLogStore(workspaceID, jobID, connID uuid.UUID) *logStore {
	return &logStore{
		jobs: map[uuid.UUID]*pipelineservice.Job{
			jobID: {ID: jobID, ConnectionID: connID, Attempt: 1, Status: pipelineservice.JobStatusRunning},
		},
		connections: map[uuid.UUID]*pipelineservice.Connection{
			connID: {ID: connID, WorkspaceID: workspaceID},
		},
		segments: map[segmentKey]*pipelineservice.JobLogSegment{},
	}
}

func (s *logStore) Jobs() pipelineservice.JobsStorage               { return &logJobs{store: s} }
func (s *logStore) Connections() pipelineservice.ConnectionsStorage { return &logConnections{store: s} }
func (s *logStore) JobLogs() pipelineservice.JobLogsStorage         { return &logLines{store: s} }
func (s *logStore) JobLogSegments() pipelineservice.JobLogSegmentsStorage {
	return &logSegments{store: s}
}

func (s *logStore) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

func (s *logStore) rowsFor(jobID uuid.UUID) []pipelineservice.JobLog {
	var out []pipelineservice.JobLog

	for _, l := range s.logs {
		if l.JobID == jobID {
			out = append(out, l)
		}
	}

	return out
}

type logJobs struct {
	pipelineservice.JobsStorage
	store *logStore
}

func (m *logJobs) First(_ context.Context, f *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Job, error) {
	rec, ok := m.store.jobs[f.ID.(*filter.EqualsFilter[uuid.UUID]).Value]
	if !ok {
		return nil, pipelineservice.ErrJobNotFound
	}

	cp := *rec

	return &cp, nil
}

type logConnections struct {
	pipelineservice.ConnectionsStorage
	store *logStore
}

func (m *logConnections) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	for _, conn := range m.store.connections {
		if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.ID != eq.Value {
			continue
		}

		if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != eq.Value {
			continue
		}

		cp := *conn

		return &cp, nil
	}

	return nil, pipelineservice.ErrConnectionNotFound
}

type logLines struct {
	pipelineservice.JobLogsStorage
	store *logStore
}

func (m *logLines) GetLogs(_ context.Context, jobID uuid.UUID, afterID int64, limit int) ([]pipelineservice.JobLog, error) {
	out := make([]pipelineservice.JobLog, 0)

	for _, l := range m.store.rowsFor(jobID) {
		if l.ID <= afterID {
			continue
		}

		out = append(out, l)
		if limit > 0 && len(out) >= limit {
			break
		}
	}

	return out, nil
}

func (m *logLines) InsertLines(_ context.Context, jobID uuid.UUID, attempt int, lines []string) error {
	for _, line := range lines {
		m.store.nextID++
		m.store.logs = append(m.store.logs, pipelineservice.JobLog{ID: m.store.nextID, JobID: jobID, Attempt: attempt, LogLine: line})
	}

	return nil
}

func (m *logLines) InsertLine(ctx context.Context, jobID uuid.UUID, attempt int, line string) (int64, error) {
	if err := m.InsertLines(ctx, jobID, attempt, []string{line}); err != nil {
		return 0, err
	}

	return m.store.nextID, nil
}

func (m *logLines) UpdateLine(_ context.Context, id int64, line string) (bool, error) {
	for i := range m.store.logs {
		if m.store.logs[i].ID == id {
			m.store.logs[i].LogLine = line

			return true, nil
		}
	}

	return false, nil
}

func (m *logLines) MaxLogID(_ context.Context, jobID uuid.UUID, attempt int) (int64, error) {
	var maxID int64

	for _, l := range m.store.rowsFor(jobID) {
		if l.Attempt == attempt {
			maxID = max(maxID, l.ID)
		}
	}

	return maxID, nil
}

func (m *logLines) ListAttemptLogs(_ context.Context, jobID uuid.UUID, attempt int, afterID, upToID int64, limit int) ([]pipelineservice.JobLog, error) {
	var out []pipelineservice.JobLog

	for _, l := range m.store.rowsFor(jobID) {
		if l.Attempt == attempt && l.ID > afterID && l.ID <= upToID {
			out = append(out, l)
			if len(out) == limit {
				break
			}
		}
	}

	return out, nil
}

func (m *logLines) DeleteAttemptLogs(_ context.Context, jobID uuid.UUID, attempt int, upToID int64) error {
	m.store.logs = slices.DeleteFunc(m.store.logs, func(l pipelineservice.JobLog) bool {
		return l.JobID == jobID && l.Attempt == attempt && l.ID <= upToID
	})

	return nil
}

func (m *logLines) ListUnarchivedAttempts(_ context.Context, jobID uuid.UUID) ([]int, error) {
	var attempts []int

	for _, l := range m.store.rowsFor(jobID) {
		segment, ok := m.store.segments[segmentKey{jobID, l.Attempt}]
		if ok && segment.Status == pipelineservice.JobLogSegmentStatusArchived {
			continue
		}

		if !slices.Contains(attempts, l.Attempt) {
			attempts = append(attempts, l.Attempt)
		}
	}

	slices.Sort(attempts)

	return attempts, nil
}

type logSegments struct {
	pipelineservice.JobLogSegmentsStorage
	store *logStore
}

func (m *logSegments) LockCurrent(ctx context.Context, jobID uuid.UUID) (*pipelineservice.JobLogSegment, error) {
	job, ok := m.store.jobs[jobID]
	if !ok {
		return nil, pipelineservice.ErrJobNotFound
	}

	return m.Lock(ctx, jobID, job.Attempt)
}

func (m *logSegments) Lock(_ context.Context, jobID uuid.UUID, attempt int) (*pipelineservice.JobLogSegment, error) {
	key := segmentKey{jobID, attempt}

	segment, ok := m.store.segments[key]
	if !ok {
		segment = &pipelineservice.JobLogSegment{JobID: jobID, Attempt: attempt, Status: pipelineservice.JobLogSegmentStatusOpen}
		m.store.segments[key] = segment
	}

	cp := *segment

	return &cp, nil
}

func (m *logSegments) Update(_ context.Context, segment *pipelineservice.JobLogSegment) (*pipelineservice.JobLogSegment, error) {
	cp := *segment
	m.store.segments[segmentKey{segment.JobID, segment.Attempt}] = &cp

	return segment, nil
}

func (m *logSegments) MarkArchived(_ context.Context, segment *pipelineservice.JobLogSegment) (bool, error) {
	if m.store.beforeMark != nil {
		m.store.beforeMark()
	}

	if err := m.store.markArchivedErr; err != nil {
		m.store.markArchivedErr = nil

		return false, err
	}

	stored, ok := m.store.segments[segmentKey{segment.JobID, segment.Attempt}]
	if !ok || stored.Status == pipelineservice.JobLogSegmentStatusArchived {
		return false, nil
	}

	stored.Status = pipelineservice.JobLogSegmentStatusArchived
	stored.ArchiveKey = segment.ArchiveKey
	stored.SizeBytes = segment.SizeBytes
	stored.FirstLogID = segment.FirstLogID
	stored.LastLogID = segment.LastLogID
	stored.ArchivedAt = segment.ArchivedAt

	return true, nil
}

func (m *logSegments) Find(_ context.Context, f *pipelineservice.JobLogSegmentFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.JobLogSegment, error) {
	jobID := f.JobID.(*filter.EqualsFilter[uuid.UUID]).Value
	status := f.Status.(*filter.EqualsFilter[pipelineservice.JobLogSegmentStatus]).Value

	var out []*pipelineservice.JobLogSegment

	for _, segment := range m.store.segments {
		if segment.JobID == jobID && segment.Status == status {
			cp := *segment
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (m *logSegments) First(_ context.Context, f *pipelineservice.JobLogSegmentFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.JobLogSegment, error) {
	key := segmentKey{
		f.JobID.(*filter.EqualsFilter[uuid.UUID]).Value,
		f.Attempt.(*filter.EqualsFilter[int]).Value,
	}

	segment, ok := m.store.segments[key]
	if !ok {
		return nil, pipelineservice.ErrJobLogSegmentNotFound
	}

	cp := *segment

	return &cp, nil
}

// countingArchive wraps a filesystem archive, counting writes and failing
// writes of chosen keys.
type countingArchive struct {
	*logarchive.Filesystem
	puts    int
	failPut map[string]bool
}

var errPutFailed = errors.New("put failed")

func newCountingArchive(t *testing.T) *countingArchive {
	t.Helper()

	fs, err := logarchive.NewFilesystem(t.TempDir())
	require.NoError(t, err)

	return &countingArchive{Filesystem: fs, failPut: map[string]bool{}}
}

func (a *countingArchive) Put(ctx context.Context, key string, r io.Reader) error {
	if a.failPut[key] {
		return errPutFailed
	}

	a.puts++

	return a.Filesystem.Put(ctx, key, r)
}

func (a *countingArchive) exists(t *testing.T, key string) bool {
	t.Helper()

	body, err := a.Open(context.Background(), key)
	if errors.Is(err, logarchive.ErrNotFound) {
		return false
	}

	require.NoError(t, err)
	require.NoError(t, body.Close())

	return true
}
