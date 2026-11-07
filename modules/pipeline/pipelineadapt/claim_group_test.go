package pipelineadapt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
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

func TestUseCaseExecutorBackend_ClaimsUseExecutorGroup(t *testing.T) {
	storage := &claimGroupStorage{}
	claimJobBundle, claimTask := newClaimGroupUseCases(storage)
	backend := NewUseCaseExecutorBackend(claimJobBundle, nil, nil, nil, nil, nil, nil, nil, nil, nil, claimTask, nil, nil, nil, nil, "Prod Cluster ACME")

	job, err := backend.ClaimJob(t.Context(), "w")
	require.NoError(t, err)
	assert.Nil(t, job)

	task, err := backend.ClaimConnectorTask(t.Context(), "w")
	require.NoError(t, err)
	assert.Nil(t, task)

	assert.Equal(t, []string{"Prod Cluster ACME"}, storage.jobGroups)
	assert.Equal(t, []string{"Prod Cluster ACME"}, storage.taskGroups)
}

func TestUseCaseExecutorBackend_HeartbeatRegistersExecutorGroup(t *testing.T) {
	storage := &claimGroupStorage{}
	backend := NewUseCaseExecutorBackend(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, pipelinejobs.NewRegisterExecutor(storage), "us")

	require.NoError(t, backend.ExecutorHeartbeat(t.Context(), pipelineexec.ExecutorInfo{ID: "executor-0", Type: pipelineservice.ExecutorTypeDocker, Group: "us"}))
	require.Len(t, storage.registered, 1)
	assert.Equal(t, "us", storage.registered[0].ExecutorGroup)
}
