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
