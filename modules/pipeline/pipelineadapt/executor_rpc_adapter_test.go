package pipelineadapt

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
)

// Compile-time check that RPCExecutorBackend implements ExecutorBackend.
var _ pipelinesync.ExecutorBackend = (*RPCExecutorBackend)(nil)

func TestRPCExecutorBackend_ImplementsInterface(t *testing.T) {
	// This test verifies the interface implementation at compile time.
	t.Log("RPCExecutorBackend implements pipelinesync.ExecutorBackend")
}

func TestRPCExecutorBackend_TokenInterceptorSetsHeader(t *testing.T) {
	// Verify that the token interceptor sets the X-Internal-Secret header.
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Internal-Secret")
		// Return a minimal valid Connect response.
		w.Header().Set("Content-Type", "application/proto")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	backend := NewRPCExecutorBackend(server.URL, "test-secret-token")
	require.NotNil(t, backend)

	// Call IsJobActive (simplest RPC) to trigger the interceptor.
	// We don't care about the result, just that the header was set.
	_, _ = backend.IsJobActive(t.Context(), "00000000-0000-0000-0000-000000000001")

	assert.Equal(t, "test-secret-token", receivedHeader)
}

func TestRPCExecutorBackend_ProtoJobTypeToDomain(t *testing.T) {
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
