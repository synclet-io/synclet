package pipelinelogs_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/pkg/logarchive"
)

type logFixture struct {
	workspaceID uuid.UUID
	connID      uuid.UUID
	jobID       uuid.UUID
	store       *logStore
	archive     *countingArchive
	metrics     *pipelinemetrics.MetricsCollector
	registry    *prometheus.Registry
}

func newLogFixture(t *testing.T) *logFixture {
	t.Helper()

	f := &logFixture{workspaceID: uuid.New(), connID: uuid.New(), jobID: uuid.New()}
	f.store = newLogStore(f.workspaceID, f.jobID, f.connID)
	f.archive = newCountingArchive(t)
	f.metrics = pipelinemetrics.NewMetricsCollector()
	f.registry = prometheus.NewPedanticRegistry()
	f.registry.MustRegister(f.metrics)

	return f
}

func (f *logFixture) append(t *testing.T, maxLines int, lines ...string) {
	t.Helper()

	uc := pipelinelogs.NewBatchAppendJobLogs(f.store, pipelineservice.Config{MaxLogLinesPerAttempt: maxLines}, f.metrics)
	require.NoError(t, uc.Execute(context.Background(), pipelinelogs.BatchAppendJobLogsParams{JobID: f.jobID, LogLines: lines}))
}

func (f *logFixture) get(t *testing.T, afterID int64, limit int) *pipelinelogs.GetJobLogResult {
	t.Helper()

	result, err := pipelinelogs.NewGetJobLog(f.store, f.archive).Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: f.workspaceID,
		JobID:       f.jobID,
		AfterID:     afterID,
		Limit:       limit,
	})
	require.NoError(t, err)

	return result
}

// pages reads the whole log with the given page size and returns each page.
func (f *logFixture) pages(t *testing.T, limit int) []*pipelinelogs.GetJobLogResult {
	t.Helper()

	var (
		out    []*pipelinelogs.GetJobLogResult
		cursor int64
	)

	for {
		page := f.get(t, cursor, limit)
		out = append(out, page)

		if !page.HasMore {
			return out
		}

		cursor = page.LastID
	}
}

func (f *logFixture) archiver(enabled bool) *pipelinelogs.ArchiveJobLogs {
	return pipelinelogs.NewArchiveJobLogs(f.store, f.archive, pipelineservice.Config{LogArchiveEnabled: enabled, LogArchiveBackend: "filesystem"}, f.metrics)
}

func (f *logFixture) finish(status pipelineservice.JobStatus) {
	f.store.jobs[f.jobID].Status = status
}

func (f *logFixture) retry() {
	f.store.jobs[f.jobID].Attempt++
}

// assertMetric compares the named metric gathered from the fixture registry
// with the expected exposition text.
func (f *logFixture) assertMetric(t *testing.T, name, expected string) {
	t.Helper()

	require.NoError(t, testutil.GatherAndCompare(f.registry, strings.NewReader(expected), name))
}

func (f *logFixture) key(attempt int) string {
	return pipelineservice.LogArchiveKey(f.workspaceID, f.connID, f.jobID, attempt)
}

func TestAppendAndGetJobLog(t *testing.T) {
	f := newLogFixture(t)

	f.append(t, 0, "line-1")
	f.append(t, 0, "line-2")

	result := f.get(t, 0, 0)
	assert.Equal(t, []string{"line-1", "line-2"}, result.Lines)
	assert.Equal(t, int64(2), result.LastID)
	assert.False(t, result.HasMore)
}

func TestAppendJobLog_UnknownJob(t *testing.T) {
	f := newLogFixture(t)

	uc := pipelinelogs.NewBatchAppendJobLogs(f.store, pipelineservice.Config{}, f.metrics)
	err := uc.Execute(context.Background(), pipelinelogs.BatchAppendJobLogsParams{JobID: uuid.New(), LogLines: []string{"x"}})
	require.ErrorIs(t, err, pipelineservice.ErrJobNotFound)
}

func TestBatchAppendJobLogs_TagsCurrentAttempt(t *testing.T) {
	f := newLogFixture(t)

	f.append(t, 0, "a", "b")
	f.retry()
	f.append(t, 0, "c")

	attempts := make([]int, 0, len(f.store.logs))
	for _, l := range f.store.logs {
		attempts = append(attempts, l.Attempt)
	}

	assert.Equal(t, []int{1, 1, 2}, attempts)
	assert.Equal(t, []string{"a", "b", "c"}, f.get(t, 0, 0).Lines)
}

func TestBatchAppendJobLogs_CapsLinesPerAttempt(t *testing.T) {
	f := newLogFixture(t)

	f.append(t, 3, "1", "2", "3", "4", "5")
	assert.Equal(t, []string{"1", "2", "3", "[synclet] log truncated: 2 further lines dropped"}, f.get(t, 0, 0).Lines)

	f.append(t, 3, "6", "7")
	assert.Equal(t, []string{"1", "2", "3", "[synclet] log truncated: 4 further lines dropped"}, f.get(t, 0, 0).Lines)

	segment := f.store.segments[segmentKey{f.jobID, 1}]
	assert.Equal(t, 3, segment.LineCount)
	assert.Equal(t, 4, segment.DroppedCount)

	f.retry()
	f.append(t, 3, "a", "b")
	assert.Equal(t, []string{"1", "2", "3", "[synclet] log truncated: 4 further lines dropped", "a", "b"}, f.get(t, 0, 0).Lines)
}

func TestBatchAppendJobLogs_CapAndDroppedCountMatchAcrossBatchSizes(t *testing.T) {
	lines := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	for _, sizes := range [][]int{{1, 1, 1, 1, 1, 1, 1, 1}, {8}, {2, 3, 3}, {4, 1, 3}, {5, 3}} {
		t.Run(fmt.Sprint(sizes), func(t *testing.T) {
			f := newLogFixture(t)

			rest := lines
			for _, size := range sizes {
				f.append(t, 4, rest[:size]...)
				rest = rest[size:]
			}

			assert.Equal(t, []string{"1", "2", "3", "4", "[synclet] log truncated: 4 further lines dropped"}, f.get(t, 0, 0).Lines)

			segment := f.store.segments[segmentKey{f.jobID, 1}]
			assert.Equal(t, 4, segment.LineCount)
			assert.Equal(t, 4, segment.DroppedCount)
		})
	}
}

func TestBatchAppendJobLogs_ReinsertsMarkerAfterItWasArchived(t *testing.T) {
	f := newLogFixture(t)

	f.append(t, 1, "1", "2")
	f.finish(pipelineservice.JobStatusFailed)
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	f.append(t, 1, "late")
	assert.Equal(t, []string{
		"1",
		"[synclet] log truncated: 1 further lines dropped",
		"[synclet] log truncated: 2 further lines dropped",
	}, f.get(t, 0, 0).Lines)
}

func TestGetJobLog_PaginationWithCursor(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a", "b", "c", "d", "e")

	pages := f.pages(t, 2)
	require.Len(t, pages, 3)
	assert.Equal(t, []string{"a", "b"}, pages[0].Lines)
	assert.Equal(t, int64(2), pages[0].LastID)
	assert.Equal(t, []string{"c", "d"}, pages[1].Lines)
	assert.Equal(t, []string{"e"}, pages[2].Lines)
	assert.False(t, pages[2].HasMore)
}

func TestGetJobLog_RefusesCrossWorkspace(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "hi")

	_, err := pipelinelogs.NewGetJobLog(f.store, f.archive).Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: uuid.New(),
		JobID:       f.jobID,
	})
	require.Error(t, err)
}

func TestGetJobLog_DefaultAndMaxLimit(t *testing.T) {
	f := newLogFixture(t)

	lines := make([]string, pipelinelogs.MaxLogLimit+10)
	for i := range lines {
		lines[i] = "line"
	}

	f.append(t, 0, lines...)

	byDefault := f.get(t, 0, 0)
	assert.Len(t, byDefault.Lines, pipelinelogs.DefaultLogLimit)
	assert.True(t, byDefault.HasMore)

	capped := f.get(t, 0, pipelinelogs.MaxLogLimit*2)
	assert.Len(t, capped.Lines, pipelinelogs.MaxLogLimit)
	assert.True(t, capped.HasMore)

	rest := f.get(t, capped.LastID, pipelinelogs.MaxLogLimit)
	assert.Len(t, rest.Lines, 10)
	assert.False(t, rest.HasMore)
}

func TestArchiveJobLogs_WritesObjectsThenDeletesRows(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a1", "a2", "a3")
	f.retry()
	f.append(t, 0, "b1", "b2")
	f.finish(pipelineservice.JobStatusCompleted)

	before := f.pages(t, 2)

	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	assert.Empty(t, f.store.logs)
	assert.True(t, f.archive.exists(t, f.key(1)))
	assert.True(t, f.archive.exists(t, f.key(2)))

	segment := f.store.segments[segmentKey{f.jobID, 1}]
	assert.Equal(t, pipelineservice.JobLogSegmentStatusArchived, segment.Status)
	require.NotNil(t, segment.ArchiveKey)
	assert.Equal(t, f.key(1), *segment.ArchiveKey)
	assert.Equal(t, int64(1), *segment.FirstLogID)
	assert.Equal(t, int64(3), *segment.LastLogID)
	assert.Positive(t, segment.SizeBytes)
	assert.NotNil(t, segment.ArchivedAt)

	assert.Equal(t, before, f.pages(t, 2), "archived reads return the same pages and cursors")
}

func TestArchiveJobLogs_CursorFromLiveTailSurvivesArchiving(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a", "b", "c")

	tail := f.get(t, 0, 0)
	f.append(t, 0, "d")
	f.finish(pipelineservice.JobStatusCompleted)
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	next := f.get(t, tail.LastID, 0)
	assert.Equal(t, []string{"d"}, next.Lines)
	assert.Equal(t, int64(4), next.LastID)
	assert.False(t, next.HasMore)
}

func TestArchiveJobLogs_NoOpUnlessEnabledAndFinished(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a")

	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID), "running job")
	f.finish(pipelineservice.JobStatusCancelled)
	require.NoError(t, f.archiver(false).Execute(context.Background(), f.jobID), "archiving disabled")
	require.NoError(t, f.archiver(true).Execute(context.Background(), uuid.New()), "unknown job")

	assert.Len(t, f.store.logs, 1)
	assert.Zero(t, f.archive.puts)
}

func TestArchiveJobLogs_RetryAfterFailedWrite(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a1", "a2")
	f.retry()
	f.append(t, 0, "b1")
	f.finish(pipelineservice.JobStatusFailed)

	f.archive.failPut[f.key(2)] = true
	require.ErrorIs(t, f.archiver(true).Execute(context.Background(), f.jobID), errPutFailed)

	assert.True(t, f.archive.exists(t, f.key(1)))
	assert.False(t, f.archive.exists(t, f.key(2)))
	assert.Equal(t, []string{"b1"}, []string{f.store.logs[0].LogLine}, "rows of the failed attempt stay")
	assert.Equal(t, []string{"a1", "a2", "b1"}, f.get(t, 0, 0).Lines, "mixed archived and live attempts read in order")

	delete(f.archive.failPut, f.key(2))
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	assert.Empty(t, f.store.logs)
	assert.Equal(t, 2, f.archive.puts, "the archived attempt is not rewritten")
	assert.Equal(t, []string{"a1", "a2", "b1"}, f.get(t, 0, 0).Lines)
}

func TestArchiveJobLogs_RetryAfterFailedMark(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a1")
	f.finish(pipelineservice.JobStatusCompleted)

	f.store.markArchivedErr = errPutFailed
	require.Error(t, f.archiver(true).Execute(context.Background(), f.jobID))

	assert.True(t, f.archive.exists(t, f.key(1)))
	assert.Len(t, f.store.logs, 1, "rows stay until the attempt is marked archived")

	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))
	assert.Empty(t, f.store.logs)

	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))
	assert.Equal(t, 2, f.archive.puts, "a finished job is not archived again")
	assert.Equal(t, []string{"a1"}, f.get(t, 0, 0).Lines)
}

func TestArchiveJobLogs_LinesAfterArchivingStayInPostgres(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a", "b")
	f.finish(pipelineservice.JobStatusCompleted)
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	f.append(t, 0, "late")
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	assert.Equal(t, []string{"late"}, []string{f.store.logs[0].LogLine})
	assert.Equal(t, []string{"a", "b", "late"}, f.get(t, 0, 0).Lines)

	pages := f.pages(t, 2)
	require.Len(t, pages, 2)
	assert.Equal(t, []string{"a", "b"}, pages[0].Lines)
	assert.Equal(t, []string{"late"}, pages[1].Lines)
}

func TestArchiveJobLogs_WritesNoObjectForDeletedJob(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a")
	f.finish(pipelineservice.JobStatusCompleted)

	f.store.beforeJobLock = func() {
		delete(f.store.jobs, f.jobID)
		delete(f.store.segments, segmentKey{f.jobID, 1})
	}
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	assert.Zero(t, f.archive.puts)
	assert.False(t, f.archive.exists(t, f.key(1)))

	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID), "a redelivered event of the deleted job")
	assert.Zero(t, f.archive.puts)
}

func TestArchiveJobLogs_KeepsObjectArchivedByConcurrentRun(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a")
	f.finish(pipelineservice.JobStatusCompleted)

	f.store.beforeMark = func() {
		f.store.beforeMark = nil
		key := f.key(1)
		_, err := f.store.JobLogSegments().MarkArchived(context.Background(), &pipelineservice.JobLogSegment{
			JobID: f.jobID, Attempt: 1, ArchiveKey: &key,
		})
		require.NoError(t, err)
	}
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	assert.True(t, f.archive.exists(t, f.key(1)), "the object of a live job stays")
}

func TestGetJobLog_ArchivedPagesAcrossAttempts(t *testing.T) {
	f := newLogFixture(t)

	for attempt := 1; attempt <= 3; attempt++ {
		lines := make([]string, 4)
		for i := range lines {
			lines[i] = fmt.Sprintf("%d-%d", attempt, i)
		}

		f.append(t, 0, lines...)

		if attempt < 3 {
			f.retry()
		}
	}

	f.finish(pipelineservice.JobStatusCompleted)
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))

	var got []string
	for _, page := range f.pages(t, 5) {
		got = append(got, page.Lines...)
	}

	assert.Equal(t, []string{"1-0", "1-1", "1-2", "1-3", "2-0", "2-1", "2-2", "2-3", "3-0", "3-1", "3-2", "3-3"}, got)

	last := f.get(t, 12, 0)
	assert.Empty(t, last.Lines)
	assert.False(t, last.HasMore)
}

func TestGetJobLog_MissingArchiveObject(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a")
	f.finish(pipelineservice.JobStatusCompleted)
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))
	require.NoError(t, f.archive.Delete(context.Background(), f.key(1)))

	_, err := pipelinelogs.NewGetJobLog(f.store, f.archive).Execute(context.Background(), pipelinelogs.GetJobLogParams{
		WorkspaceID: f.workspaceID,
		JobID:       f.jobID,
	})
	require.ErrorIs(t, err, logarchive.ErrNotFound)
}

func TestBatchAppendJobLogs_CountsDroppedLines(t *testing.T) {
	f := newLogFixture(t)

	f.append(t, 3, "1", "2", "3", "4", "5")
	f.append(t, 3, "6", "7")
	f.append(t, 0, "8")

	f.assertMetric(t, "synclet_log_lines_dropped_total", `
# HELP synclet_log_lines_dropped_total Total number of job log lines dropped by the per-attempt line cap.
# TYPE synclet_log_lines_dropped_total counter
synclet_log_lines_dropped_total 4
`)
}

func TestArchiveJobLogs_CountsArchivedBytes(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a1", "a2", "a3")
	f.retry()
	f.append(t, 0, "b1", "b2")
	f.finish(pipelineservice.JobStatusCompleted)

	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID))
	require.NoError(t, f.archiver(true).Execute(context.Background(), f.jobID), "rerun archives nothing")

	size := f.store.segments[segmentKey{f.jobID, 1}].SizeBytes + f.store.segments[segmentKey{f.jobID, 2}].SizeBytes
	require.Positive(t, size)

	f.assertMetric(t, "synclet_log_archive_bytes_total", fmt.Sprintf(`
# HELP synclet_log_archive_bytes_total Total size in bytes of the job log objects written to the log archive.
# TYPE synclet_log_archive_bytes_total counter
synclet_log_archive_bytes_total %d
`, size))
}

func TestArchiveJobLogs_CountsFailuresByBackend(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a1")
	f.finish(pipelineservice.JobStatusFailed)

	f.archive.failPut[f.key(1)] = true
	require.ErrorIs(t, f.archiver(true).Execute(context.Background(), f.jobID), errPutFailed)
	require.ErrorIs(t, f.archiver(true).Execute(context.Background(), f.jobID), errPutFailed)

	f.assertMetric(t, "synclet_log_archive_failures_total", `
# HELP synclet_log_archive_failures_total Total number of job attempts whose logs failed to archive, by archive backend.
# TYPE synclet_log_archive_failures_total counter
synclet_log_archive_failures_total{backend="filesystem"} 2
`)
	f.assertMetric(t, "synclet_log_archive_bytes_total", `
# HELP synclet_log_archive_bytes_total Total size in bytes of the job log objects written to the log archive.
# TYPE synclet_log_archive_bytes_total counter
synclet_log_archive_bytes_total 0
`)
}

func (f *logFixture) assertArchiveMetrics(t *testing.T, failures int, bytes int64) {
	t.Helper()

	f.assertMetric(t, "synclet_log_archive_failures_total", fmt.Sprintf(`
# HELP synclet_log_archive_failures_total Total number of job attempts whose logs failed to archive, by archive backend.
# TYPE synclet_log_archive_failures_total counter
synclet_log_archive_failures_total{backend="filesystem"} %d
`, failures))
	f.assertMetric(t, "synclet_log_archive_bytes_total", fmt.Sprintf(`
# HELP synclet_log_archive_bytes_total Total size in bytes of the job log objects written to the log archive.
# TYPE synclet_log_archive_bytes_total counter
synclet_log_archive_bytes_total %d
`, bytes))
}

func TestArchiveJobLogs_FailureBeforeAttemptLoopCountsNoAttempt(t *testing.T) {
	errStorage := errors.New("storage unavailable")

	for name, breakStore := range map[string]func(f *logFixture){
		"connection lookup":           func(f *logFixture) { f.store.connectionErr = errStorage },
		"listing unarchived attempts": func(f *logFixture) { f.store.listAttemptsErr = errStorage },
	} {
		t.Run(name, func(t *testing.T) {
			f := newLogFixture(t)
			f.metrics.InitLogArchiveFailures("filesystem")
			f.append(t, 0, "a1")
			f.finish(pipelineservice.JobStatusCompleted)
			breakStore(f)

			require.ErrorIs(t, f.archiver(true).Execute(context.Background(), f.jobID), errStorage)

			assert.Zero(t, f.archive.puts)
			f.assertArchiveMetrics(t, 0, 0)
		})
	}
}

func TestArchiveJobLogs_CancelledRunCountsFailedAttemptOnceAndCommittedBytesOnly(t *testing.T) {
	f := newLogFixture(t)
	f.append(t, 0, "a1", "a2")
	f.retry()
	f.append(t, 0, "b1")
	f.finish(pipelineservice.JobStatusCompleted)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	locks := 0
	f.store.beforeJobLock = func() {
		locks++
		if locks == 2 {
			cancel()
		}
	}

	require.ErrorIs(t, f.archiver(true).Execute(ctx, f.jobID), context.Canceled)

	assert.True(t, f.archive.exists(t, f.key(1)))
	assert.False(t, f.archive.exists(t, f.key(2)))
	f.assertArchiveMetrics(t, 1, f.store.segments[segmentKey{f.jobID, 1}].SizeBytes)
}
