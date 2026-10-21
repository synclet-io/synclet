package pipelineexec

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

type nopHandler struct {
	Handler
}

type discardWriteCloser struct{}

func (discardWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (discardWriteCloser) Close() error                { return nil }

const recordLine = `{"type":"RECORD","record":{"stream":"users","data":{"id":1},"emitted_at":0}}` + "\n"

func runSilentSource(ctx context.Context, t *testing.T, firstLine string, idleTimeout time.Duration) error {
	t.Helper()

	srcReader, srcWriter := io.Pipe()
	destReader, destWriter := io.Pipe()

	t.Cleanup(func() {
		_ = srcWriter.Close()
		_ = destWriter.Close()
	})

	if firstLine != "" {
		go func() { _, _ = io.WriteString(srcWriter, firstLine) }()
	}

	type outcome struct {
		stats *Stats
		err   error
	}

	result := make(chan outcome, 1)

	go func() {
		stats, err := Run(ctx, srcReader, discardWriteCloser{}, destReader, nopHandler{}, RunConfig{IdleTimeout: idleTimeout}, nil)
		result <- outcome{stats: stats, err: err}
	}()

	select {
	case res := <-result:
		if firstLine != "" {
			assert.Equal(t, int64(1), res.stats.RecordsRead)
		}

		return res.err
	case <-time.After(5 * time.Second):
		t.Fatal("router did not return while the source was silent")

		return nil
	}
}

func TestRun_IdleTimeoutFiresWhileSourceIsSilent(t *testing.T) {
	err := runSilentSource(context.Background(), t, recordLine, 50*time.Millisecond)

	require.ErrorContains(t, err, "idle timeout")
}

func TestRun_IdleTimeoutWaitsForFirstRecord(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := runSilentSource(ctx, t, "", 20*time.Millisecond)

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRun_ContextCancelStopsSilentSource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := runSilentSource(ctx, t, recordLine, 0)

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRun_RejectsSourceLineOverSyncLimit(t *testing.T) {
	line := `{"type":"RECORD","record":{"stream":"users","data":{"blob":"` +
		strings.Repeat("x", protocol.SyncMessageMaxSize) + `"},"emitted_at":0}}` + "\n"

	_, err := Run(context.Background(), strings.NewReader(line), discardWriteCloser{}, io.NopCloser(strings.NewReader("")), nopHandler{}, RunConfig{}, nil)

	require.ErrorIs(t, err, protocol.ErrMessageTooLarge)
	assert.ErrorContains(t, err, "16 MiB")
}

func TestRun_RejectsDestinationLineOverSyncLimit(t *testing.T) {
	line := `{"type":"LOG","log":{"level":"INFO","message":"` +
		strings.Repeat("x", protocol.SyncMessageMaxSize) + `"}}` + "\n"

	_, err := Run(context.Background(), strings.NewReader(""), discardWriteCloser{}, io.NopCloser(strings.NewReader(line)), nopHandler{}, RunConfig{}, nil)

	require.ErrorIs(t, err, protocol.ErrMessageTooLarge)
	assert.ErrorContains(t, err, "16 MiB")
}
