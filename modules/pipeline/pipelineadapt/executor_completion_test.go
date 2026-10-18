package pipelineadapt

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
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

// completionStorage serves one running job and records the job that
// UpdateJobStatus writes.
type completionStorage struct {
	pipelineservice.Storage
	job     *pipelineservice.Job
	written *pipelineservice.Job
}

func (s *completionStorage) Jobs() pipelineservice.JobsStorage { return &completionJobs{s: s} }

func (s *completionStorage) Connections() pipelineservice.ConnectionsStorage {
	return &completionConnections{}
}

func (s *completionStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type completionJobs struct {
	pipelineservice.JobsStorage
	s *completionStorage
}

func (j *completionJobs) First(_ context.Context, _ *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Job, error) {
	cp := *j.s.job

	return &cp, nil
}

func (j *completionJobs) UpdateIfStatus(_ context.Context, job *pipelineservice.Job, _ ...pipelineservice.JobStatus) (bool, error) {
	j.s.written = job

	return false, nil
}

type completionConnections struct {
	pipelineservice.ConnectionsStorage
}

func (completionConnections) First(_ context.Context, _ *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	return nil, nil //nolint:nilnil // no connection skips retention cleanup
}

func TestUseCaseExecutorBackend_ReportCompletion_ClassifiesExitCodes(t *testing.T) {
	tests := []struct {
		name       string
		sourceExit int
		destExit   int
	}{
		{name: "source OOM", sourceExit: 137},
		{name: "destination OOM", destExit: 137},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			storage := &completionStorage{job: &pipelineservice.Job{
				ID:          uuid.New(),
				Status:      pipelineservice.JobStatusRunning,
				Attempt:     1,
				MaxAttempts: 3,
			}}
			logger := &logging.Logger{Delegate: noopLogDelegate{}}
			backend := &UseCaseExecutorBackend{
				reportCompletion: pipelinejobs.NewReportCompletion(pipelinejobs.NewUpdateJobStatus(storage), storage, nil, logger, pipelineservice.Config{}),
			}

			err := backend.ReportCompletion(t.Context(), pipelineexec.ReportCompletionParams{
				JobID:          storage.job.ID,
				ConnectionID:   uuid.New(),
				ErrorMessage:   "connector exited with code 137",
				SourceExitCode: testCase.sourceExit,
				DestExitCode:   testCase.destExit,
			})
			require.NoError(t, err)

			require.NotNil(t, storage.written)
			assert.Equal(t, pipelineservice.JobStatusFailed, storage.written.Status)
			require.NotNil(t, storage.written.FailureReason)
			assert.Equal(t, "OOMKilled (exit 137)", *storage.written.FailureReason)
		})
	}
}
