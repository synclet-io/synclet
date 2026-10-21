package pipelineexec

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/logtest"
)

// lockedLogHandler records log lines from the source and destination readers,
// which call OnLog concurrently.
type lockedLogHandler struct {
	DefaultHandler

	mu    sync.Mutex
	lines []string
}

func (h *lockedLogHandler) OnLog(_ context.Context, line string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lines = append(h.lines, line)

	return nil
}

func TestRun_RedactsConnectorTextInProcessLogs(t *testing.T) {
	const secret = "hunter2-router-secret"

	longMessage := strings.Repeat("x", MaxLogLineSize-4) + secret + strings.Repeat("y", 1<<10)
	source := strings.Join([]string{
		`{"type":"LOG","log":{"level":"INFO","message":"src ` + secret + `"}}`,
		`{"type":"LOG","log":{"level":"` + secret + `","message":"odd level"}}`,
		`{"type":"LOG","log":{"level":"INFO","message":"` + longMessage + `"}}`,
		`{"type":"TRACE","trace":{"type":"ANALYTICS","emitted_at":0,"analytics":{"type":"` + secret + `","value":"` + secret + `"}}}`,
	}, "\n") + "\n"
	dest := `{"type":"LOG","log":{"level":"WARN","message":"dst ` + secret + `"}}` + "\n" +
		`{"type":"LOG","log":{"level":"` + secret + `","message":"odd level"}}` + "\n" +
		`{"type":"TRACE","trace":{"type":"ANALYTICS","emitted_at":0,"analytics":{"type":"auth","value":"` + secret + `"}}}` + "\n"

	var logs logtest.Recorder

	handler := &lockedLogHandler{}

	_, err := Run(context.Background(), strings.NewReader(source), discardWriteCloser{}, io.NopCloser(strings.NewReader(dest)), handler, RunConfig{
		Redactor: NewRedactor(json.RawMessage(`{"api_key":"` + secret + `"}`)),
	}, logs.Logger())
	require.NoError(t, err)

	output := logs.String()
	for _, want := range []string{"source log", "dest log", "source analytics trace", "dest analytics trace", "bytes truncated"} {
		assert.Contains(t, output, want)
	}

	assert.NotContains(t, output, secret)
	assert.NotContains(t, output, "hunt")
	assert.Less(t, len(output), 2*MaxLogLineSize)

	require.NotEmpty(t, handler.lines)
	assert.NotContains(t, strings.Join(handler.lines, "\n"), secret)
}

func TestRun_RedactsSecretsFromConnectorConfigUpdates(t *testing.T) {
	const (
		srcRotated  = "rotated-source-token"
		destRotated = "rotated-dest-token"
	)

	control := func(token string) string {
		return `{"type":"CONTROL","control":{"type":"CONNECTOR_CONFIG","emitted_at":0,"connectorConfig":{"config":{"access_token":"` + token + `"}}}}`
	}
	source := control(srcRotated) + "\n" +
		`{"type":"LOG","log":{"level":"INFO","message":"refreshed to ` + srcRotated + `"}}` + "\n"
	dest := control(destRotated) + "\n" +
		`{"type":"LOG","log":{"level":"INFO","message":"refreshed to ` + destRotated + `"}}` + "\n"

	var logs logtest.Recorder

	handler := &lockedLogHandler{}

	_, err := Run(context.Background(), strings.NewReader(source), discardWriteCloser{}, io.NopCloser(strings.NewReader(dest)), handler, RunConfig{
		Redactor: NewRedactor(json.RawMessage(`{"access_token":"initial-token"}`)),
	}, logs.Logger())
	require.NoError(t, err)

	output := logs.String()
	require.Contains(t, output, "source log")
	require.Contains(t, output, "dest log")
	assert.NotContains(t, output, srcRotated)
	assert.NotContains(t, output, destRotated)

	lines := strings.Join(handler.lines, "\n")
	require.Len(t, handler.lines, 2)
	assert.NotContains(t, lines, srcRotated)
	assert.NotContains(t, lines, destRotated)
}
