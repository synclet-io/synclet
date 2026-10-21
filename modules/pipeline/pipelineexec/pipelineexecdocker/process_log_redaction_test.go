package pipelineexecdocker

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/pkg/logtest"
)

const processLogSecret = "hunter2-process-log-secret"

// runLeakySync runs a sync whose source and destination echo the source
// config secret in every kind of output and returns the process logs.
func runLeakySync(t *testing.T, sourceExit int) string {
	t.Helper()

	runner := &pairRunner{
		sourceOutput: strings.Join([]string{
			`{"type":"LOG","log":{"level":"INFO","message":"connecting with ` + processLogSecret + `"}}`,
			`not json ` + processLogSecret,
			`{"type":"TRACE","trace":{"type":"ANALYTICS","emitted_at":0,"analytics":{"type":"auth","value":"` + processLogSecret + `"}}}`,
			`{"type":"TRACE","trace":{"type":"ERROR","emitted_at":0,"error":{"message":"login failed for ` + processLogSecret + `","failure_type":"config_error"}}}`,
		}, "\n") + "\n",
		sourceStderr: "Traceback: password=" + processLogSecret,
		sourceExit:   sourceExit,
		destOutput: `{"type":"LOG","log":{"level":"WARN","message":"dest saw ` + processLogSecret + `"}}` + "\n" +
			`{"type":"TRACE","trace":{"type":"ANALYTICS","emitted_at":0,"analytics":{"type":"auth","value":"` + processLogSecret + `"}}}` + "\n",
		destDone: make(chan struct{}),
	}

	var logs logtest.Recorder

	backend := &mockBackend{}
	worker := &DockerSyncWorker{
		backend: backend,
		metrics: pipelinemetrics.NewMetricsCollector(),
		executor: NewDockerSyncExecutor(DockerSyncExecutorParams{
			Client:      NewConnectorClient(runner),
			Reporter:    backend,
			TempDirRoot: t.TempDir(),
			Logger:      logs.Logger(),
		}),
		maxSyncDuration:   time.Minute,
		heartbeatInterval: time.Minute,
		semaphore:         make(chan struct{}, 1),
		workerID:          "test-worker",
		logger:            logs.Logger(),
	}

	worker.executeJob(context.Background(), &pipelineexec.ClaimJobResult{
		Job:               &pipelineservice.Job{ID: uuid.New(), JobType: pipelineservice.JobTypeSync},
		SourceImage:       "source:1",
		DestImage:         "destination:1",
		SourceConfig:      json.RawMessage(`{"password":"` + processLogSecret + `"}`),
		DestConfig:        json.RawMessage(`{}`),
		ConfiguredCatalog: []byte(`{"streams":[]}`),
	})

	return logs.String()
}

func TestDockerSyncWorker_ProcessLogsNeverCarryConfigSecrets(t *testing.T) {
	output := runLeakySync(t, 0)

	require.Contains(t, output, "source log")
	require.Contains(t, output, "dest log")
	assert.Contains(t, output, pipelineexec.RedactedMask)
	assert.NotContains(t, output, processLogSecret)
}

func TestDockerSyncWorker_FailureLogNeverCarriesStderrSecrets(t *testing.T) {
	output := runLeakySync(t, 1)

	require.Contains(t, output, "job failed")
	assert.Contains(t, output, "Traceback: password="+pipelineexec.RedactedMask)
	assert.NotContains(t, output, processLogSecret)
}
