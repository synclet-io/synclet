package pipelineexecdocker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// pairRunner runs a source that prints sourceOutput and exits with
// sourceExit, and a destination that records everything it reads.
type pairRunner struct {
	mockContainerRunner

	sourceOutput string
	sourceExit   int

	mu       sync.Mutex
	runs     int
	destRead bytes.Buffer
}

func (r *pairRunner) Run(_ context.Context, opts pkgcontainer.RunOptions) (*pkgcontainer.RunResult, error) {
	r.mu.Lock()
	r.runs++
	isSource := r.runs == 1
	r.mu.Unlock()

	done := make(chan struct{})

	if isSource {
		close(done)

		return &pkgcontainer.RunResult{
			Stdout:   io.NopCloser(strings.NewReader(r.sourceOutput)),
			Stderr:   io.NopCloser(strings.NewReader("")),
			ExitCode: r.sourceExit,
			Done:     done,
		}, nil
	}

	stdoutReader, stdoutWriter := io.Pipe()

	go func() {
		input, _ := io.ReadAll(opts.Stdin)

		r.mu.Lock()
		r.destRead.Write(input)
		r.mu.Unlock()

		close(done)

		_ = stdoutWriter.Close()
	}()

	return &pkgcontainer.RunResult{
		Stdout: stdoutReader,
		Stderr: io.NopCloser(strings.NewReader("")),
		Done:   done,
	}, nil
}

func (r *pairRunner) destinationStreamStatuses(t *testing.T) []protocol.AirbyteStreamStatusTraceMessage {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	var statuses []protocol.AirbyteStreamStatusTraceMessage

	for _, line := range strings.Split(strings.TrimSpace(r.destRead.String()), "\n") {
		var msg protocol.AirbyteMessage
		require.NoError(t, json.Unmarshal([]byte(line), &msg), line)

		if msg.Trace != nil && msg.Trace.StreamStatus != nil {
			statuses = append(statuses, *msg.Trace.StreamStatus)
		}
	}

	return statuses
}

func executeWithSource(t *testing.T, sourceExit int) (*pairRunner, error) {
	t.Helper()

	runner := &pairRunner{
		sourceOutput: `{"type":"RECORD","record":{"stream":"users","namespace":"public","data":{"id":1},"emitted_at":0}}` + "\n",
		sourceExit:   sourceExit,
	}

	prefix := "dst_"
	executor := NewDockerSyncExecutor(DockerSyncExecutorParams{
		Client:      NewConnectorClient(runner),
		TempDirRoot: t.TempDir(),
		Logger:      &logging.Logger{Delegate: discardLog{}},
	})

	_, err := executor.Execute(context.Background(), &pipelineexec.SyncBundle{
		Job:                 &pipelineservice.Job{ID: uuid.New()},
		SourceImage:         "source:1",
		DestImage:           "destination:1",
		SourceConfig:        json.RawMessage(`{}`),
		DestConfig:          json.RawMessage(`{}`),
		NamespaceDefinition: pipelineservice.NamespaceDefinitionSource,
		StreamPrefix:        &prefix,
		ConfiguredCatalog: &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
			Stream:              protocol.AirbyteStream{Name: "users", Namespace: "public", JSONSchema: json.RawMessage(`{"type":"object"}`)},
			SyncMode:            protocol.SyncModeFullRefresh,
			DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
		}}},
	}, func() {})

	return runner, err
}

func TestDockerSyncExecutor_CompletesStreamsAfterSourceExitsZero(t *testing.T) {
	runner, err := executeWithSource(t, 0)
	require.NoError(t, err)

	assert.Equal(t, []protocol.AirbyteStreamStatusTraceMessage{
		{StreamDescriptor: protocol.StreamDescriptor{Name: "dst_users", Namespace: "public"}, Status: protocol.StreamStatusComplete},
	}, runner.destinationStreamStatuses(t))
}

func TestDockerSyncExecutor_SendsNoStatusAfterSourceFails(t *testing.T) {
	runner, err := executeWithSource(t, 1)
	require.Error(t, err)

	assert.Empty(t, runner.destinationStreamStatuses(t))
}
