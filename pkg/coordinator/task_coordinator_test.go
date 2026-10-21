package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
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

type recordingTaskServer struct {
	executorv1connect.UnimplementedExecutorServiceHandler

	reports chan *executorv1.ReportConnectorTaskResultRequest
}

func (s *recordingTaskServer) ReportConnectorTaskResult(_ context.Context, req *connect.Request[executorv1.ReportConnectorTaskResultRequest]) (*connect.Response[executorv1.ReportConnectorTaskResultResponse], error) {
	s.reports <- req.Msg

	return connect.NewResponse(&executorv1.ReportConnectorTaskResultResponse{}), nil
}

func serveRecordedTasks(t *testing.T) (addr string, reports <-chan *executorv1.ReportConnectorTaskResultRequest) {
	t.Helper()

	srv := &recordingTaskServer{reports: make(chan *executorv1.ReportConnectorTaskResultRequest, 1)}
	_, handler := executorv1connect.NewExecutorServiceHandler(srv)
	httpSrv := httptest.NewServer(handler)
	t.Cleanup(httpSrv.Close)

	return httpSrv.URL, srv.reports
}

func runTaskWithin(t *testing.T, cfg TaskConfig, limit time.Duration) error {
	t.Helper()

	done := make(chan error, 1)

	go func() { done <- RunTask(t.Context(), cfg) }()

	select {
	case err := <-done:
		return err
	case <-time.After(limit):
		t.Fatalf("RunTask did not return within %s", limit)

		return nil
	}
}

func TestRunTask_ReportsAFailureWhenConnectorOutputCannotBeRead(t *testing.T) {
	fastTaskReportRetry(t)

	addr, reports := serveRecordedTasks(t)
	dataDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "connector.exitcode"), []byte("0\n"), 0o600))
	// A directory opens fine but every read fails, like an unreadable stream.
	require.NoError(t, os.Mkdir(filepath.Join(dataDir, "connector-stdout.log"), 0o700))

	err := runTaskWithin(t, TaskConfig{TaskID: "t1", TaskType: "Discover", ServerAddr: addr, DataDir: dataDir}, 10*time.Second)
	require.Error(t, err)

	report := <-reports
	assert.False(t, report.GetSuccess())
	assert.Contains(t, report.GetErrorMessage(), "reading connector output")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("disk gone") }

func TestReadTaskMessages_StopsAtTheFirstReadError(t *testing.T) {
	_, err := readTaskMessages(failingReader{})

	require.ErrorContains(t, err, "disk gone")
}

func catalogLine(descriptionSize int) string {
	return `{"type":"CATALOG","catalog":{"streams":[{"name":"wide","json_schema":{"description":"` +
		strings.Repeat("d", descriptionSize) + `"}}]}}` + "\n"
}

func TestReadTaskMessages_AcceptsCatalogUnderTaskLimit(t *testing.T) {
	messages, err := readTaskMessages(strings.NewReader(catalogLine(protocol.TaskMessageMaxSize - 1<<20)))

	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, protocol.MessageTypeCatalog, messages[0].Type)
}

func TestReadTaskMessages_RejectsCatalogOverTaskLimit(t *testing.T) {
	_, err := readTaskMessages(strings.NewReader(catalogLine(protocol.TaskMessageMaxSize)))

	require.ErrorIs(t, err, protocol.ErrMessageTooLarge)
	assert.ErrorContains(t, err, "50 MiB")
}

func TestRunTask_TimeoutReportsTheTailOfConnectorOutput(t *testing.T) {
	fastTaskReportRetry(t)

	saved := connectorTaskTimeout
	connectorTaskTimeout = 300 * time.Millisecond

	t.Cleanup(func() { connectorTaskTimeout = saved })

	addr, reports := serveRecordedTasks(t)
	dataDir := t.TempDir()
	secretsDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(secretsDir, "task-config"), []byte(`{"api_key":"airtable-secret-42"}`), 0o600))

	output := strings.Repeat("early noise line\n", 2000) + "Traceback: auth with airtable-secret-42 hangs\n"
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "connector-stdout.log"), []byte(output), 0o600))

	err := runTaskWithin(t, TaskConfig{TaskID: "t1", TaskType: "Discover", ServerAddr: addr, DataDir: dataDir, SecretsDir: secretsDir}, 10*time.Second)
	require.Error(t, err)

	report := <-reports
	msg := report.GetErrorMessage()
	assert.False(t, report.GetSuccess())
	assert.Contains(t, msg, "timed out")
	assert.Contains(t, msg, "Traceback: auth with")
	assert.NotContains(t, msg, "airtable-secret-42")
	assert.Less(t, len(msg), 2*connectorOutputTailSize)
}

func TestConnectorOutputTail_MasksASecretCutByTheTail(t *testing.T) {
	const secret = "airtable-secret-42"

	redactor := pipelineexec.NewRedactor(json.RawMessage(`{"api_key":"` + secret + `"}`))

	tests := map[string]string{
		"secret straddling the tail start": strings.Repeat("x", 10000) + secret + strings.Repeat("y", connectorOutputTailSize-len(secret)/2),
		"output cut off inside a secret":   "Traceback\nconnecting with " + secret[:12],
	}

	for name, output := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "connector-stdout.log")
			require.NoError(t, os.WriteFile(path, []byte(output), 0o600))

			tail := connectorOutputTail(path, connectorOutputTailSize, redactor)

			assert.NotEmpty(t, tail)
			assert.LessOrEqual(t, len(tail), connectorOutputTailSize)
			assert.NotContains(t, tail, "secret-42")
			assert.NotContains(t, tail, "airtable-sec")
		})
	}
}
