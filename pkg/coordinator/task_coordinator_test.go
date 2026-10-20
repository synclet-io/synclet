package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestTaskConfigRedactor_MasksCheckStatusMessage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task-config"), []byte(`{"password":"task-secret-9"}`), 0o600))

	redactor := taskConfigRedactor(dir)
	assert.NotContains(t, redactor.Redact("auth failed for task-secret-9"), "task-secret-9")

	result, err := extractTaskResult("Check", []*protocol.AirbyteMessage{{
		Type: protocol.MessageTypeConnectionStatus,
		ConnectionStatus: &protocol.AirbyteConnectionStatus{
			Status:  protocol.ConnectionStatusFailed,
			Message: "login rejected: task-secret-9",
		},
	}}, redactor)
	require.NoError(t, err)

	var decoded struct{ Message string }
	require.NoError(t, json.Unmarshal(result, &decoded))
	assert.NotContains(t, decoded.Message, "task-secret-9")
}

func TestTaskConfigRedactor_WithoutConfigRedactsNothing(t *testing.T) {
	assert.Nil(t, taskConfigRedactor(""))
	assert.Nil(t, taskConfigRedactor(t.TempDir()))
}

type flakyTaskServer struct {
	executorv1connect.UnimplementedExecutorServiceHandler

	failures int
	code     connect.Code
	calls    atomic.Int32
}

func (s *flakyTaskServer) ReportConnectorTaskResult(context.Context, *connect.Request[executorv1.ReportConnectorTaskResultRequest]) (*connect.Response[executorv1.ReportConnectorTaskResultResponse], error) {
	if int(s.calls.Add(1)) <= s.failures {
		return nil, connect.NewError(s.code, errors.New("not now"))
	}

	return connect.NewResponse(&executorv1.ReportConnectorTaskResultResponse{}), nil
}

func serveTasks(t *testing.T, srv *flakyTaskServer) string {
	t.Helper()

	_, handler := executorv1connect.NewExecutorServiceHandler(srv)
	httpSrv := httptest.NewServer(handler)
	t.Cleanup(httpSrv.Close)

	return httpSrv.URL
}

func fastTaskReportRetry(t *testing.T) {
	t.Helper()

	saved := taskReportRetry
	taskReportRetry = retryPolicy{initial: time.Millisecond, max: 5 * time.Millisecond, total: 2 * time.Second}

	t.Cleanup(func() { taskReportRetry = saved })
}

func TestReportTaskResult_RetriesUntilTheServerAnswers(t *testing.T) {
	fastTaskReportRetry(t)

	srv := &flakyTaskServer{failures: 3, code: connect.CodeUnavailable}

	err := reportTaskResult(TaskConfig{TaskID: "t1", ServerAddr: serveTasks(t, srv)}, true, "", []byte("{}"))

	require.NoError(t, err)
	assert.Equal(t, int32(4), srv.calls.Load())
}

func TestReportTaskResult_DoesNotRetryARejectedResult(t *testing.T) {
	fastTaskReportRetry(t)

	srv := &flakyTaskServer{failures: 100, code: connect.CodeInvalidArgument}

	err := reportTaskResult(TaskConfig{TaskID: "t1", ServerAddr: serveTasks(t, srv)}, true, "", []byte("{}"))

	require.Error(t, err)
	assert.Equal(t, int32(1), srv.calls.Load())
}

func TestReportTaskResult_GivesUpAfterTheRetryWindow(t *testing.T) {
	fastTaskReportRetry(t)

	srv := &flakyTaskServer{failures: 1 << 20, code: connect.CodeUnavailable}

	err := reportTaskResult(TaskConfig{TaskID: "t1", ServerAddr: serveTasks(t, srv)}, true, "", []byte("{}"))

	require.Error(t, err)
	assert.Greater(t, srv.calls.Load(), int32(1))
}
