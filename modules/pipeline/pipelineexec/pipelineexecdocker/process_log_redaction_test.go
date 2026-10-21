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

	output, _ := runSync(t, runner, json.RawMessage(`{"password":"`+processLogSecret+`"}`))

	return output
}

// runSync runs a sync of runner's connectors with sourceConfig and returns
// the process logs and the backend the job was reported to.
func runSync(t *testing.T, runner *pairRunner, sourceConfig json.RawMessage) (string, *mockBackend) {
	t.Helper()

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
		SourceConfig:      sourceConfig,
		DestConfig:        json.RawMessage(`{}`),
		ConfiguredCatalog: []byte(`{"streams":[]}`),
	})

	return logs.String(), backend
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

func TestDockerSyncWorker_FailureNeverCarriesSecretRotatedThroughControl(t *testing.T) {
	const rotated = "rotated-access-token-value"

	runner := &pairRunner{
		sourceOutput: `{"type":"CONTROL","control":{"type":"CONNECTOR_CONFIG","emitted_at":0,"connectorConfig":{"config":{"access_token":"` + rotated + `"}}}}` + "\n",
		sourceStderr: "Traceback: token " + rotated + " expired",
		sourceExit:   1,
		destDone:     make(chan struct{}),
	}

	output, backend := runSync(t, runner, json.RawMessage(`{"access_token":"`+processLogSecret+`"}`))

	require.Equal(t, int32(1), backend.completionCalled.Load())
	require.False(t, backend.lastCompletion.Success)
	assert.Contains(t, backend.lastCompletion.ErrorMessage, "Traceback: token "+pipelineexec.RedactedMask+" expired")
	assert.NotContains(t, backend.lastCompletion.ErrorMessage, rotated)
	require.Contains(t, output, "job failed")
	assert.NotContains(t, output, rotated)
}
