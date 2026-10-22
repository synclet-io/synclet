package pipelineexecdocker

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

type taskReportBackend struct {
	pipelineexec.ExecutorBackend

	reports []pipelineexec.ReportConnectorTaskResultParams
}

func (b *taskReportBackend) ReportConnectorTaskResult(_ context.Context, params pipelineexec.ReportConnectorTaskResultParams) error {
	b.reports = append(b.reports, params)

	return nil
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestDockerTaskWorkerRedactsTaskErrors(t *testing.T) {
	const secret = "hunter2-pass"

	config := []byte(`{"host":"db.internal","password":"` + secret + `"}`)

	closedDone := func() chan struct{} {
		done := make(chan struct{})
		close(done)

		return done
	}

	tests := []struct {
		name     string
		taskType pipelineservice.ConnectorTaskType
		runner   *mockContainerRunner
	}{
		{
			name:     "check start error",
			taskType: pipelineservice.ConnectorTaskTypeCheck,
			runner:   &mockContainerRunner{runErrors: []error{errors.New("creating container: env PASSWORD=" + secret)}},
		},
		{
			name:     "discover pull error",
			taskType: pipelineservice.ConnectorTaskTypeDiscover,
			runner: &mockContainerRunner{
				runErrors: []error{errors.New("No such image: img")},
				pullError: errors.New("registry rejected credentials " + secret),
			},
		},
		{
			name:     "discover message read error",
			taskType: pipelineservice.ConnectorTaskTypeDiscover,
			runner: &mockContainerRunner{runResults: []*pkgcontainer.RunResult{{
				Stdout: io.NopCloser(failingReader{err: errors.New("malformed line {\"password\":\"" + secret + "\"}")}),
				Stderr: io.NopCloser(failingReader{err: io.EOF}),
				Done:   closedDone(),
			}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &taskReportBackend{}
			worker := &DockerTaskWorker{backend: backend, client: NewConnectorClient(tt.runner)}

			worker.executeTask(context.Background(), &pipelineexec.ClaimConnectorTaskResult{
				TaskID:   uuid.New(),
				TaskType: tt.taskType,
				Image:    "img",
				Config:   config,
			})

			require.Len(t, backend.reports, 1)
			report := backend.reports[0]
			assert.False(t, report.Success)
			assert.NotEmpty(t, report.ErrorMessage)
			assert.NotContains(t, report.ErrorMessage, secret)
			assert.Contains(t, report.ErrorMessage, pipelineexec.RedactedMask)
		})
	}
}

func TestDockerTaskWorkerAppliesResources(t *testing.T) {
	taskTypes := map[string]pipelineservice.ConnectorTaskType{
		"check":    pipelineservice.ConnectorTaskTypeCheck,
		"spec":     pipelineservice.ConnectorTaskTypeSpec,
		"discover": pipelineservice.ConnectorTaskTypeDiscover,
	}

	tests := []struct {
		name          string
		defaults      pipelineservice.RuntimeDefaults
		runtimeConfig string
		want          ResourceLimits
	}{
		{name: "nothing configured"},
		{
			name:          "runtime_config",
			runtimeConfig: `{"memory_request":"256Mi","cpu_request":"1","memory_limit":"1Gi","cpu_limit":"2"}`,
			want:          ResourceLimits{MemoryRequest: 268435456, CPURequest: 1, MemoryLimit: 1073741824, CPULimit: 2},
		},
		{
			name:     "runtime defaults",
			defaults: pipelineservice.RuntimeDefaults{MemoryRequest: "512Mi", CPURequest: "500m"},
			want:     ResourceLimits{MemoryRequest: 536870912, CPURequest: 0.5},
		},
	}

	for typeName, taskType := range taskTypes {
		for _, tt := range tests {
			t.Run(typeName+"/"+tt.name, func(t *testing.T) {
				runner := &mockContainerRunner{runErrors: []error{errors.New("boom")}}
				worker := &DockerTaskWorker{backend: &taskReportBackend{}, client: NewConnectorClient(runner), runtimeDefaults: tt.defaults}

				worker.executeTask(context.Background(), &pipelineexec.ClaimConnectorTaskResult{
					TaskID:        uuid.New(),
					TaskType:      taskType,
					Image:         "img",
					Config:        []byte(`{}`),
					RuntimeConfig: tt.runtimeConfig,
				})

				require.Equal(t, 1, runner.runCalls)
				opts := runner.lastRunOpts
				assert.Equal(t, tt.want.MemoryRequest, opts.MemoryRequest, "MemoryRequest")
				assert.InDelta(t, tt.want.CPURequest, opts.CPURequest, 1e-9, "CPURequest")
				assert.Equal(t, tt.want.MemoryLimit, opts.MemoryLimit, "MemoryLimit")
				assert.InDelta(t, tt.want.CPULimit, opts.CPULimit, 1e-9, "CPULimit")
			})
		}
	}
}
