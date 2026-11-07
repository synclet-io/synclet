package pipelineexecrpc

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// claimRecorder answers every claim with nothing to claim and keeps the
// executor group each claim names.
type claimRecorder struct {
	executorv1connect.UnimplementedExecutorServiceHandler
	jobGroups       []string
	taskGroups      []string
	heartbeatGroups []string
}

func (r *claimRecorder) ExecutorHeartbeat(_ context.Context, req *connect.Request[executorv1.ExecutorHeartbeatRequest]) (*connect.Response[executorv1.ExecutorHeartbeatResponse], error) {
	r.heartbeatGroups = append(r.heartbeatGroups, req.Msg.GetExecutorGroup())

	return connect.NewResponse(&executorv1.ExecutorHeartbeatResponse{}), nil
}

func (r *claimRecorder) ClaimJob(_ context.Context, req *connect.Request[executorv1.ClaimJobRequest]) (*connect.Response[executorv1.ClaimJobResponse], error) {
	r.jobGroups = append(r.jobGroups, req.Msg.GetExecutorGroup())

	return connect.NewResponse(&executorv1.ClaimJobResponse{}), nil
}

func (r *claimRecorder) ClaimConnectorTask(_ context.Context, req *connect.Request[executorv1.ClaimConnectorTaskRequest]) (*connect.Response[executorv1.ClaimConnectorTaskResponse], error) {
	r.taskGroups = append(r.taskGroups, req.Msg.GetExecutorGroup())

	return connect.NewResponse(&executorv1.ClaimConnectorTaskResponse{}), nil
}

func TestBackend_ClaimsCarryExecutorGroup(t *testing.T) {
	recorder := &claimRecorder{}
	_, handler := executorv1connect.NewExecutorServiceHandler(recorder)

	server := httptest.NewServer(handler)
	defer server.Close()

	backend := NewBackend(server.URL, "token", "Prod Cluster ACME")

	job, err := backend.ClaimJob(t.Context(), "worker")
	require.NoError(t, err)
	assert.Nil(t, job)

	task, err := backend.ClaimConnectorTask(t.Context(), "worker")
	require.NoError(t, err)
	assert.Nil(t, task)

	assert.Equal(t, []string{"Prod Cluster ACME"}, recorder.jobGroups, "ClaimJob names the executor's group")
	assert.Equal(t, []string{"Prod Cluster ACME"}, recorder.taskGroups, "ClaimConnectorTask names the executor's group")
}

func TestBackend_ExecutorHeartbeatCarriesExecutorGroup(t *testing.T) {
	recorder := &claimRecorder{}
	_, handler := executorv1connect.NewExecutorServiceHandler(recorder)

	server := httptest.NewServer(handler)
	defer server.Close()

	require.NoError(t, NewBackend(server.URL, "token", "us").ExecutorHeartbeat(t.Context(), pipelineexec.ExecutorInfo{
		ID: "executor-0", Type: pipelineservice.ExecutorTypeDocker, Group: "us",
	}))
	assert.Equal(t, []string{"us"}, recorder.heartbeatGroups)
}
