package pipelineexecrpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// Compile-time check that Backend implements ExecutorBackend.
var _ pipelineexec.ExecutorBackend = (*Backend)(nil)

func TestBackend_ImplementsInterface(t *testing.T) {
	// This test verifies the interface implementation at compile time.
	t.Log("Backend implements pipelineexec.ExecutorBackend")
}

func TestBackend_TokenInterceptorSetsHeader(t *testing.T) {
	// Verify that the token interceptor sets the X-Internal-Secret header.
	var receivedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Internal-Secret")
		// Return a minimal valid Connect response.
		w.Header().Set("Content-Type", "application/proto")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	backend := NewBackend(server.URL, "test-secret-token")
	require.NotNil(t, backend)

	// Call IsJobActive (simplest RPC) to trigger the interceptor.
	// We don't care about the result, just that the header was set.
	_, _ = backend.IsJobActive(t.Context(), "00000000-0000-0000-0000-000000000001")

	assert.Equal(t, "test-secret-token", receivedHeader)
}

type completionRecorder struct {
	executorv1connect.UnimplementedExecutorServiceHandler
	got *executorv1.ReportCompletionRequest
}

func (r *completionRecorder) ReportCompletion(_ context.Context, req *connect.Request[executorv1.ReportCompletionRequest]) (*connect.Response[executorv1.ReportCompletionResponse], error) {
	r.got = req.Msg

	return connect.NewResponse(&executorv1.ReportCompletionResponse{}), nil
}

func TestBackend_ReportCompletionSendsExitCodes(t *testing.T) {
	recorder := &completionRecorder{}
	_, handler := executorv1connect.NewExecutorServiceHandler(recorder)

	server := httptest.NewServer(handler)
	defer server.Close()

	backend := NewBackend(server.URL, "token")
	err := backend.ReportCompletion(t.Context(), pipelineexec.ReportCompletionParams{
		JobID:          uuid.New(),
		ConnectionID:   uuid.New(),
		ErrorMessage:   "destination connector exited with code 137",
		SourceExitCode: 1,
		DestExitCode:   137,
	})
	require.NoError(t, err)

	require.NotNil(t, recorder.got)
	assert.Equal(t, int32(1), recorder.got.GetSourceExitCode())
	assert.Equal(t, int32(137), recorder.got.GetDestExitCode())
}

func TestBackend_ProtoJobTypeToDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    executorv1.JobType
		expected pipelineservice.JobType
	}{
		{"sync", executorv1.JobType_JOB_TYPE_SYNC, pipelineservice.JobTypeSync},
		{"unspecified defaults to sync", executorv1.JobType_JOB_TYPE_UNSPECIFIED, pipelineservice.JobTypeSync},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protoJobTypeToDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackend_SendWithRetry(t *testing.T) {
	backend := &Backend{retryBaseDelay: time.Millisecond, retryMaxDelay: time.Millisecond}

	t.Run("stops on non-retryable code", func(t *testing.T) {
		calls := 0
		err := backend.sendWithRetry(t.Context(), "UpdateJobStatus", func(context.Context) error {
			calls++

			return connect.NewError(connect.CodeNotFound, errors.New("job not found"))
		})

		require.Error(t, err)
		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		assert.Equal(t, 1, calls)
	})

	t.Run("caps transient retries", func(t *testing.T) {
		calls := 0
		err := backend.sendWithRetry(t.Context(), "Heartbeat", func(context.Context) error {
			calls++

			return connect.NewError(connect.CodeUnavailable, errors.New("server down"))
		})

		require.Error(t, err)
		assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
		assert.Equal(t, rpcMaxAttempts, calls)
	})

	t.Run("succeeds after transient failure", func(t *testing.T) {
		calls := 0
		err := backend.sendWithRetry(t.Context(), "ClaimJob", func(context.Context) error {
			calls++
			if calls < 3 {
				return connect.NewError(connect.CodeUnavailable, errors.New("server down"))
			}

			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, calls)
	})
}

type connectorTaskServer struct {
	executorv1connect.UnimplementedExecutorServiceHandler
	resp *executorv1.ClaimConnectorTaskResponse
}

func (s *connectorTaskServer) ClaimConnectorTask(_ context.Context, _ *connect.Request[executorv1.ClaimConnectorTaskRequest]) (*connect.Response[executorv1.ClaimConnectorTaskResponse], error) {
	return connect.NewResponse(s.resp), nil
}

func TestBackend_ClaimConnectorTaskCarriesRuntimeSettings(t *testing.T) {
	_, handler := executorv1connect.NewExecutorServiceHandler(&connectorTaskServer{resp: &executorv1.ClaimConnectorTaskResponse{
		HasTask:            true,
		TaskId:             uuid.NewString(),
		WorkspaceId:        uuid.NewString(),
		Image:              "source:1",
		ServiceAccountName: "wi-sa",
		RuntimeConfig:      `{"memory_request":"256Mi"}`,
	}})

	server := httptest.NewServer(handler)
	defer server.Close()

	got, err := NewBackend(server.URL, "token").ClaimConnectorTask(t.Context(), "worker")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "wi-sa", got.ServiceAccountName)
	assert.JSONEq(t, `{"memory_request":"256Mi"}`, got.RuntimeConfig)
}
