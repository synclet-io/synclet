package pipelineexec

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const (
	usersRecordLine = `{"type":"RECORD","record":{"stream":"users","namespace":"public","data":{"id":1},"emitted_at":0}}`
	finalStateLine  = `{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"c":1}}}}`
)

func streamStatusLine(name, namespace string, status protocol.StreamStatus) string {
	msg := protocol.AirbyteMessage{
		Type: protocol.MessageTypeTrace,
		Trace: &protocol.AirbyteTraceMessage{
			Type:      protocol.TraceTypeStreamStatus,
			EmittedAt: 1,
			StreamStatus: &protocol.AirbyteStreamStatusTraceMessage{
				StreamDescriptor: protocol.StreamDescriptor{Name: name, Namespace: namespace},
				Status:           status,
			},
		},
	}

	line, _ := json.Marshal(msg)

	return string(line)
}

func usersOrdersCatalog() *protocol.ConfiguredAirbyteCatalog {
	return makeCatalog(makeStream("users", "public"), makeStream("orders", "sales"))
}

func routeToDestination(t *testing.T, source io.Reader, cfg RunConfig) ([]protocol.AirbyteMessage, error) {
	t.Helper()

	dest := &bufferWriteCloser{}
	_, err := Run(context.Background(), source, dest, io.NopCloser(strings.NewReader("")), nopHandler{}, cfg, nil)

	var written []protocol.AirbyteMessage

	scanner := bufio.NewScanner(&dest.Buffer)
	for scanner.Scan() {
		var msg protocol.AirbyteMessage
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &msg), scanner.Text())

		written = append(written, msg)
	}

	return written, err
}

func streamStatuses(msgs []protocol.AirbyteMessage) []protocol.AirbyteStreamStatusTraceMessage {
	var statuses []protocol.AirbyteStreamStatusTraceMessage

	for _, msg := range msgs {
		if msg.Type == protocol.MessageTypeTrace && msg.Trace != nil && msg.Trace.StreamStatus != nil {
			statuses = append(statuses, *msg.Trace.StreamStatus)
		}
	}

	return statuses
}

func TestRun_CompletesEveryStreamAfterSuccessfulSource(t *testing.T) {
	customFmt := "custom_${SOURCE_NAMESPACE}"
	catalog := usersOrdersCatalog()
	source := usersRecordLine + "\n" + finalStateLine + "\n"

	before := time.Now().UnixMilli()
	written, err := routeToDestination(t, strings.NewReader(source), RunConfig{
		Catalog:  catalog,
		Rewriter: NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionCustom, &customFmt, strPtr("p_")),
	})
	require.NoError(t, err)

	require.Len(t, written, 4)
	assert.Equal(t, protocol.MessageTypeRecord, written[0].Type)
	assert.Equal(t, protocol.MessageTypeState, written[1].Type)

	for _, msg := range written[2:] {
		require.Equal(t, protocol.MessageTypeTrace, msg.Type)
		require.Equal(t, protocol.TraceTypeStreamStatus, msg.Trace.Type)
		assert.GreaterOrEqual(t, msg.Trace.EmittedAt, float64(before))
		assert.LessOrEqual(t, msg.Trace.EmittedAt, float64(time.Now().UnixMilli()))
	}

	assert.ElementsMatch(t, []protocol.AirbyteStreamStatusTraceMessage{
		{StreamDescriptor: protocol.StreamDescriptor{Name: "p_users", Namespace: "custom_public"}, Status: protocol.StreamStatusComplete},
		{StreamDescriptor: protocol.StreamDescriptor{Name: "p_orders", Namespace: "custom_sales"}, Status: protocol.StreamStatusComplete},
	}, streamStatuses(written))
}

func TestRun_CompletesOnlyStreamsWithoutTerminalStatus(t *testing.T) {
	catalog := makeCatalog(makeStream("users", "public"), makeStream("orders", "sales"), makeStream("events", ""))
	source := strings.Join([]string{
		streamStatusLine("users", "public", protocol.StreamStatusStarted),
		usersRecordLine,
		streamStatusLine("users", "public", protocol.StreamStatusComplete),
		streamStatusLine("orders", "sales", protocol.StreamStatusIncomplete),
		streamStatusLine("events", "", protocol.StreamStatusRunning),
		finalStateLine,
	}, "\n") + "\n"

	written, err := routeToDestination(t, strings.NewReader(source), RunConfig{
		Catalog:  catalog,
		Rewriter: NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionSource, nil, strPtr("dst_")),
	})
	require.NoError(t, err)

	assert.Equal(t, []protocol.AirbyteStreamStatusTraceMessage{
		{StreamDescriptor: protocol.StreamDescriptor{Name: "dst_users", Namespace: "public"}, Status: protocol.StreamStatusStarted},
		{StreamDescriptor: protocol.StreamDescriptor{Name: "dst_users", Namespace: "public"}, Status: protocol.StreamStatusComplete},
		{StreamDescriptor: protocol.StreamDescriptor{Name: "dst_orders", Namespace: "sales"}, Status: protocol.StreamStatusIncomplete},
		{StreamDescriptor: protocol.StreamDescriptor{Name: "dst_events"}, Status: protocol.StreamStatusRunning},
		{StreamDescriptor: protocol.StreamDescriptor{Name: "dst_events"}, Status: protocol.StreamStatusComplete},
	}, streamStatuses(written))
}

type failingReader struct {
	data io.Reader
	err  error
}

func (r *failingReader) Read(p []byte) (int, error) {
	n, err := r.data.Read(p)
	if errors.Is(err, io.EOF) {
		return n, r.err
	}

	return n, err
}

func TestRun_SendsNoStatusWhenSourceFails(t *testing.T) {
	source := usersRecordLine + "\n" + finalStateLine + "\n"

	t.Run("read error", func(t *testing.T) {
		written, err := routeToDestination(t, &failingReader{data: strings.NewReader(source), err: errors.New("source exited with code 1")}, RunConfig{Catalog: usersOrdersCatalog()})
		require.Error(t, err)
		assert.Empty(t, streamStatuses(written))
		assert.Len(t, written, 2)
	})

	t.Run("non-zero exit after clean output", func(t *testing.T) {
		written, err := routeToDestination(t, strings.NewReader(source), RunConfig{
			Catalog:         usersOrdersCatalog(),
			SourceSucceeded: func() bool { return false },
		})
		require.NoError(t, err)
		assert.Empty(t, streamStatuses(written))
		assert.Len(t, written, 2)
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		srcReader, srcWriter := io.Pipe()
		dest := &bufferWriteCloser{}

		done := make(chan error, 1)

		go func() {
			_, err := Run(ctx, srcReader, dest, io.NopCloser(strings.NewReader("")), nopHandler{}, RunConfig{Catalog: usersOrdersCatalog()}, nil)
			done <- err
		}()

		_, err := io.WriteString(srcWriter, usersRecordLine+"\n")
		require.NoError(t, err)
		cancel()

		require.ErrorIs(t, <-done, context.Canceled)

		_ = srcWriter.Close()

		assert.NotContains(t, dest.String(), "STREAM_STATUS")
	})

	t.Run("idle timeout", func(t *testing.T) {
		srcReader, srcWriter := io.Pipe()

		t.Cleanup(func() { _ = srcWriter.Close() })

		go func() { _, _ = io.WriteString(srcWriter, usersRecordLine+"\n") }()

		dest := &bufferWriteCloser{}
		_, err := Run(context.Background(), srcReader, dest, io.NopCloser(strings.NewReader("")), nopHandler{}, RunConfig{
			Catalog:     usersOrdersCatalog(),
			IdleTimeout: 50 * time.Millisecond,
		}, nil)
		require.ErrorContains(t, err, "idle timeout")
		assert.NotContains(t, dest.String(), "STREAM_STATUS")
	})
}
