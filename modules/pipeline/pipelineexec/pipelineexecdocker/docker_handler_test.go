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

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// fakeReporter records each reported log batch and, like a call to the
// control plane, fails when its context is done.
type fakeReporter struct {
	mu      sync.Mutex
	batches [][]string
}

func (f *fakeReporter) ReportState(context.Context, uuid.UUID, uuid.UUID, *protocol.AirbyteStateMessage) error {
	return nil
}

func (f *fakeReporter) ReportConfigUpdate(context.Context, pipelineservice.ConnectorType, uuid.UUID, []byte) error {
	return nil
}

func (f *fakeReporter) ReportLog(ctx context.Context, _ uuid.UUID, lines []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.batches = append(f.batches, slices.Clone(lines))

	return nil
}

func (f *fakeReporter) snapshot() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.batches)
}

func (f *fakeReporter) lines() []string {
	var all []string
	for _, batch := range f.snapshot() {
		all = append(all, batch...)
	}

	return all
}

func newTestHandler(reporter *fakeReporter, batchSize int, interval time.Duration) *DockerHandler {
	return NewDockerHandler(DockerHandlerParams{
		JobID:            uuid.New(),
		Reporter:         reporter,
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
	reporter := &fakeReporter{}
	h := newTestHandler(reporter, 3, time.Hour)
	stop := h.StartLogFlush(context.Background())

	for _, line := range numberedLines(7) {
		require.NoError(t, h.OnLog(context.Background(), line))
	}

	assert.Equal(t, [][]string{{"line-0", "line-1", "line-2"}, {"line-3", "line-4", "line-5"}}, reporter.snapshot())

	stop()

	assert.Equal(t, [][]string{{"line-0", "line-1", "line-2"}, {"line-3", "line-4", "line-5"}, {"line-6"}}, reporter.snapshot())
}

func TestDockerHandler_FlushesLogsOnInterval(t *testing.T) {
	reporter := &fakeReporter{}
	h := newTestHandler(reporter, 100, 10*time.Millisecond)

	stop := h.StartLogFlush(context.Background())
	defer stop()

	require.NoError(t, h.OnLog(context.Background(), "a"))
	require.NoError(t, h.OnLog(context.Background(), "b"))

	require.Eventually(t, func() bool {
		return slices.Equal(reporter.lines(), []string{"a", "b"})
	}, time.Second, 5*time.Millisecond)
	assert.Equal(t, [][]string{{"a", "b"}}, reporter.snapshot())
}

func TestDockerHandler_FlushesBufferedLogsWhenAttemptEnds(t *testing.T) {
	reporter := &fakeReporter{}
	h := newTestHandler(reporter, 100, time.Hour)
	stop := h.StartLogFlush(context.Background())

	require.NoError(t, h.OnLog(context.Background(), "a"))
	require.NoError(t, h.OnLog(context.Background(), "b"))
	assert.Empty(t, reporter.snapshot())

	stop()

	assert.Equal(t, [][]string{{"a", "b"}}, reporter.snapshot())
}

func TestDockerHandler_FlushesLogsAfterCancel(t *testing.T) {
	reporter := &fakeReporter{}
	h := newTestHandler(reporter, 2, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	stop := h.StartLogFlush(ctx)

	require.NoError(t, h.OnLog(ctx, "a"))
	cancel()
	require.NoError(t, h.OnLog(ctx, "b"))
	require.NoError(t, h.OnLog(ctx, "c"))

	stop()

	assert.Equal(t, [][]string{{"a", "b"}, {"c"}}, reporter.snapshot())
}

func TestDockerHandler_PreservesLogOrder(t *testing.T) {
	reporter := &fakeReporter{}
	h := newTestHandler(reporter, 7, time.Millisecond)
	stop := h.StartLogFlush(context.Background())

	lines := numberedLines(2000)
	for _, line := range lines {
		require.NoError(t, h.OnLog(context.Background(), line))
	}

	stop()

	assert.Equal(t, lines, reporter.lines())
	assert.Greater(t, len(reporter.snapshot()), 1)
}
