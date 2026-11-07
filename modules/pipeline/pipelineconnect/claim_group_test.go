package pipelineconnect

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

// claimGroupStorage has nothing to claim and keeps the executor group each
// claim reaches storage with.
type claimGroupStorage struct {
	pipelineservice.Storage
	jobGroups  []string
	taskGroups []string
	registered []*pipelineservice.Executor
}

func (s *claimGroupStorage) Executors() pipelineservice.ExecutorsStorage {
	return claimGroupExecutors{s: s}
}

type claimGroupExecutors struct {
	pipelineservice.ExecutorsStorage
	s *claimGroupStorage
}

func (e claimGroupExecutors) Upsert(_ context.Context, executor *pipelineservice.Executor) error {
	e.s.registered = append(e.s.registered, executor)

	return nil
}

func (s *claimGroupStorage) Jobs() pipelineservice.JobsStorage { return claimGroupJobs{s: s} }

func (s *claimGroupStorage) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return claimGroupTasks{s: s}
}

type claimGroupJobs struct {
	pipelineservice.JobsStorage
	s *claimGroupStorage
}

func (j claimGroupJobs) ClaimNextScheduledJob(_ context.Context, _, executorGroup string) (*pipelineservice.Job, error) {
	j.s.jobGroups = append(j.s.jobGroups, executorGroup)

	return nil, nil
}

type claimGroupTasks struct {
	pipelineservice.ConnectorTasksStorage
	s *claimGroupStorage
}

func (t claimGroupTasks) ClaimPendingTask(_ context.Context, _, executorGroup string) (*pipelineservice.ConnectorTask, error) {
	t.s.taskGroups = append(t.s.taskGroups, executorGroup)

	return nil, nil
}

func newClaimGroupUseCases(storage *claimGroupStorage) (*pipelinejobs.ClaimJobBundle, *pipelinetasks.ClaimTask) {
	return pipelinejobs.NewClaimJobBundle(pipelinejobs.NewClaimJob(storage, nil), nil, nil, nil, nil, storage, nil, nil),
		pipelinetasks.NewClaimTask(storage, nil)
}

func TestExecutorHandler_ClaimsUseExecutorGroup(t *testing.T) {
	storage := &claimGroupStorage{}
	claimJobBundle, claimTask := newClaimGroupUseCases(storage)
	handler := &ExecutorHandler{claimJobBundle: claimJobBundle, claimTask: claimTask}

	_, err := handler.ClaimJob(t.Context(), connect.NewRequest(&executorv1.ClaimJobRequest{WorkerId: "w", ExecutorGroup: "us"}))
	require.NoError(t, err)

	_, err = handler.ClaimConnectorTask(t.Context(), connect.NewRequest(&executorv1.ClaimConnectorTaskRequest{WorkerId: "w", ExecutorGroup: "us"}))
	require.NoError(t, err)

	_, err = handler.ClaimJob(t.Context(), connect.NewRequest(&executorv1.ClaimJobRequest{WorkerId: "w"}))
	require.NoError(t, err)

	assert.Equal(t, []string{"us", pipelineservice.DefaultExecutorGroup}, storage.jobGroups, "a job claim uses the requested group; none means default")
	assert.Equal(t, []string{"us"}, storage.taskGroups, "a task claim uses the requested group")
}

func TestExecutorHandler_HeartbeatRegistersExecutorGroup(t *testing.T) {
	storage := &claimGroupStorage{}
	handler := &ExecutorHandler{registerExecutor: pipelinejobs.NewRegisterExecutor(storage)}

	for _, group := range []string{" us ", ""} {
		_, err := handler.ExecutorHeartbeat(t.Context(), connect.NewRequest(&executorv1.ExecutorHeartbeatRequest{
			ExecutorId: "executor-0", ExecutorType: pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER, ExecutorGroup: group,
		}))
		require.NoError(t, err)
	}

	require.Len(t, storage.registered, 2)
	assert.Equal(t, "us", storage.registered[0].ExecutorGroup)
	assert.Equal(t, pipelineservice.DefaultExecutorGroup, storage.registered[1].ExecutorGroup, "no group means default")
}
