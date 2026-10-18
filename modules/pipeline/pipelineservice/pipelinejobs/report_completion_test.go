package pipelinejobs

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type noopLogDelegate struct{}

func (noopLogDelegate) Info(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopLogDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopLogDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopLogDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopLogDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopLogDelegate) Named(string) logging.Delegate                      { return d }
func (d noopLogDelegate) SkipCallers(int) logging.Delegate                   { return d }

func reportFailure(t *testing.T, storage *jobStateStorage, params ReportCompletionParams) {
	t.Helper()

	settings := new(mockSettingsProvider)
	settings.On("Execute", mock.Anything, mock.Anything).Return(&pipelineservice.WorkspaceSettings{}, nil)

	uc := NewReportCompletion(
		NewUpdateJobStatus(storage),
		storage,
		NewCleanupOldJobs(storage, nil, settings, nil, nil),
		&logging.Logger{Delegate: noopLogDelegate{}},
		pipelineservice.Config{},
	)
	t.Cleanup(uc.Wait)

	params.JobID = storage.job.ID
	params.ConnectionID = storage.conn.ID
	require.NoError(t, uc.Execute(context.Background(), params))
}

func TestReportCompletion_OOMKilledSourceIsNotRetried(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)

	reportFailure(t, storage, ReportCompletionParams{
		ErrorMessage:   "source connector exited with code 137",
		SourceExitCode: 137,
	})

	assert.Equal(t, pipelineservice.JobStatusFailed, storage.job.Status)
	assert.Equal(t, 1, storage.job.Attempt)
	require.NotNil(t, storage.job.FailureReason)
	assert.Equal(t, "OOMKilled (exit 137)", *storage.job.FailureReason)
	require.NotNil(t, storage.job.Error)
	assert.Equal(t, "source connector exited with code 137", *storage.job.Error)
}

func TestReportCompletion_OOMKilledDestinationIsNotRetriedWhenSourceAlsoFailed(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)

	reportFailure(t, storage, ReportCompletionParams{
		ErrorMessage:   "destination connector exited with code 137",
		SourceExitCode: 1,
		DestExitCode:   137,
	})

	assert.Equal(t, pipelineservice.JobStatusFailed, storage.job.Status)
	require.NotNil(t, storage.job.FailureReason)
	assert.Equal(t, "OOMKilled (exit 137)", *storage.job.FailureReason)
}

func TestReportCompletion_TransientExitIsRetried(t *testing.T) {
	storage := newJobStateStorage(pipelineservice.JobStatusRunning)

	reportFailure(t, storage, ReportCompletionParams{
		ErrorMessage:   "source connector exited with code 1",
		SourceExitCode: 1,
	})

	assert.Equal(t, pipelineservice.JobStatusScheduled, storage.job.Status)
	assert.Equal(t, 2, storage.job.Attempt)
	require.NotNil(t, storage.job.FailureReason)
	assert.Equal(t, "exit code 1", *storage.job.FailureReason)
}
