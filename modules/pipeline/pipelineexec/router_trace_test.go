package pipelineexec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakerStreamStatusLines are STREAM_STATUS traces as airbyte/source-faker:6.2.38 prints them.
var fakerStreamStatusLines = []string{
	`{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1792357541682.079,"stream_status":{"stream_descriptor":{"name":"users"},"status":"STARTED"}}}`,
	`{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1792357541699.031,"stream_status":{"stream_descriptor":{"name":"users"},"status":"RUNNING"}}}`,
	`{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1792357541701.0232,"stream_status":{"stream_descriptor":{"name":"users"},"status":"COMPLETE"}}}`,
}

type bufferWriteCloser struct{ bytes.Buffer }

func (*bufferWriteCloser) Close() error { return nil }

func TestRun_ForwardsStreamStatusTraceWithEveryField(t *testing.T) {
	source := strings.Join(fakerStreamStatusLines, "\n") + "\n"
	dest := &bufferWriteCloser{}

	_, err := Run(context.Background(), strings.NewReader(source), dest, io.NopCloser(strings.NewReader("")), nopHandler{}, RunConfig{}, nil)
	require.NoError(t, err)

	var forwarded []string

	scanner := bufio.NewScanner(&dest.Buffer)
	for scanner.Scan() {
		forwarded = append(forwarded, scanner.Text())
	}

	require.Len(t, forwarded, len(fakerStreamStatusLines))

	for i, want := range fakerStreamStatusLines {
		assert.Equal(t, decodeJSON(t, want), decodeJSON(t, forwarded[i]), "trace %d", i)
	}
}

type logRecordingHandler struct {
	DefaultHandler
	lines []string
}

func (h *logRecordingHandler) OnLog(_ context.Context, line string) error {
	h.lines = append(h.lines, line)

	return nil
}

func TestRun_KeepsEstimateTraceOutOfDestination(t *testing.T) {
	const (
		estimate     = `{"type":"TRACE","trace":{"type":"ESTIMATE","emitted_at":1792357541682,"estimate":{"name":"users","type":"STREAM","row_estimate":3,"byte_estimate":1350}}}`
		streamStatus = `{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1792357541682.079,"stream_status":{"stream_descriptor":{"name":"users"},"status":"STARTED"}}}`
		errorTrace   = `{"type":"TRACE","trace":{"type":"ERROR","emitted_at":1792357541700,"error":{"message":"boom","failure_type":"system_error","stream_descriptor":{"name":"users"}}}}`
	)

	source := strings.Join([]string{streamStatus, estimate, errorTrace}, "\n") + "\n"
	dest := &bufferWriteCloser{}
	handler := &logRecordingHandler{}

	_, err := Run(context.Background(), strings.NewReader(source), dest, io.NopCloser(strings.NewReader("")), handler, RunConfig{}, nil)
	require.NoError(t, err)

	var forwarded []any

	scanner := bufio.NewScanner(&dest.Buffer)
	for scanner.Scan() {
		forwarded = append(forwarded, decodeJSON(t, scanner.Text()))
	}

	assert.Equal(t, []any{decodeJSON(t, streamStatus), decodeJSON(t, errorTrace)}, forwarded)
	assert.Contains(t, handler.lines, "[src] TRACE: ESTIMATE")
}

func decodeJSON(t *testing.T, line string) any {
	t.Helper()

	var v any
	require.NoError(t, json.Unmarshal([]byte(line), &v), line)

	return v
}
