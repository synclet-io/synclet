package pipelineexecdocker

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
)

// fakeLogAppender records each appended batch and, like a database call,
// fails when its context is done.
type fakeLogAppender struct {
	mu      sync.Mutex
	batches [][]string
}

func (f *fakeLogAppender) Execute(ctx context.Context, params pipelinelogs.BatchAppendJobLogsParams) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.batches = append(f.batches, slices.Clone(params.LogLines))

	return nil
}

func (f *fakeLogAppender) snapshot() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.batches)
}

func (f *fakeLogAppender) lines() []string {
	var all []string
	for _, batch := range f.snapshot() {
		all = append(all, batch...)
	}

	return all
}

func newTestHandler(appender *fakeLogAppender, batchSize int, interval time.Duration) *DockerHandler {
	return NewDockerHandler(DockerHandlerParams{
		JobID:            uuid.New(),
		AppendJobLogs:    appender,
		LogBatchSize:     batchSize,
		LogFlushInterval: interval,
	})
}

func numberedLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line-%d", i)
	}

	return lines
}

func TestDockerHandler_FlushesLogsWhenBatchIsFull(t *testing.T) {
	appender := &fakeLogAppender{}
	h := newTestHandler(appender, 3, time.Hour)
	stop := h.StartLogFlush(context.Background())

	for _, line := range numberedLines(7) {
		require.NoError(t, h.OnLog(context.Background(), line))
	}

	assert.Equal(t, [][]string{{"line-0", "line-1", "line-2"}, {"line-3", "line-4", "line-5"}}, appender.snapshot())

	stop()

	assert.Equal(t, [][]string{{"line-0", "line-1", "line-2"}, {"line-3", "line-4", "line-5"}, {"line-6"}}, appender.snapshot())
}

func TestDockerHandler_FlushesLogsOnInterval(t *testing.T) {
	appender := &fakeLogAppender{}
	h := newTestHandler(appender, 100, 10*time.Millisecond)

	stop := h.StartLogFlush(context.Background())
	defer stop()

	require.NoError(t, h.OnLog(context.Background(), "a"))
	require.NoError(t, h.OnLog(context.Background(), "b"))

	require.Eventually(t, func() bool {
		return slices.Equal(appender.lines(), []string{"a", "b"})
	}, time.Second, 5*time.Millisecond)
	assert.Equal(t, [][]string{{"a", "b"}}, appender.snapshot())
}

func TestDockerHandler_FlushesBufferedLogsWhenAttemptEnds(t *testing.T) {
	appender := &fakeLogAppender{}
	h := newTestHandler(appender, 100, time.Hour)
	stop := h.StartLogFlush(context.Background())

	require.NoError(t, h.OnLog(context.Background(), "a"))
	require.NoError(t, h.OnLog(context.Background(), "b"))
	assert.Empty(t, appender.snapshot())

	stop()

	assert.Equal(t, [][]string{{"a", "b"}}, appender.snapshot())
}

func TestDockerHandler_FlushesLogsAfterCancel(t *testing.T) {
	appender := &fakeLogAppender{}
	h := newTestHandler(appender, 2, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	stop := h.StartLogFlush(ctx)

	require.NoError(t, h.OnLog(ctx, "a"))
	cancel()
	require.NoError(t, h.OnLog(ctx, "b"))
	require.NoError(t, h.OnLog(ctx, "c"))

	stop()

	assert.Equal(t, [][]string{{"a", "b"}, {"c"}}, appender.snapshot())
}

func TestDockerHandler_PreservesLogOrder(t *testing.T) {
	appender := &fakeLogAppender{}
	h := newTestHandler(appender, 7, time.Millisecond)
	stop := h.StartLogFlush(context.Background())

	lines := numberedLines(2000)
	for _, line := range lines {
		require.NoError(t, h.OnLog(context.Background(), line))
	}

	stop()

	assert.Equal(t, lines, appender.lines())
	assert.Greater(t, len(appender.snapshot()), 1)
}
