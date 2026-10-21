package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/pkg/protocol"
)

type taskResultRecorder struct {
	executorv1connect.UnimplementedExecutorServiceHandler

	resultSize int
}

func (r *taskResultRecorder) ReportConnectorTaskResult(_ context.Context, req *connect.Request[executorv1.ReportConnectorTaskResultRequest]) (*connect.Response[executorv1.ReportConnectorTaskResultResponse], error) {
	r.resultSize = len(req.Msg.GetResult())

	return connect.NewResponse(&executorv1.ReportConnectorTaskResultResponse{}), nil
}

func serveExecutorHandler(t *testing.T, handler executorv1connect.ExecutorServiceHandler) string {
	t.Helper()

	path, h := newExecutorServiceHandler(handler, "token")
	mux := http.NewServeMux()
	mux.Handle(path, h)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv.URL
}

func TestExecutorHandler_AcceptsTaskResultOfAFullCatalog(t *testing.T) {
	recorder := &taskResultRecorder{}
	url := serveExecutorHandler(t, recorder)

	client := executorv1connect.NewExecutorServiceClient(http.DefaultClient, url)
	req := connect.NewRequest(&executorv1.ReportConnectorTaskResultRequest{
		TaskId:  "t1",
		Success: true,
		Result:  bytes.Repeat([]byte(`\"`), protocol.TaskMessageMaxSize),
	})
	req.Header().Set("X-Internal-Secret", "token")

	_, err := client.ReportConnectorTaskResult(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, 2*protocol.TaskMessageMaxSize, recorder.resultSize)
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)

	return len(p), nil
}

func TestExecutorHandler_RejectsRequestOverReadLimit(t *testing.T) {
	recorder := &taskResultRecorder{}
	url := serveExecutorHandler(t, recorder)

	size := int64(executorRequestMaxBytes) + 1
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		url+executorv1connect.ExecutorServiceReportConnectorTaskResultProcedure, io.LimitReader(zeroReader{}, size))
	require.NoError(t, err)

	req.ContentLength = size
	req.Header.Set("Content-Type", "application/proto")
	req.Header.Set("X-Internal-Secret", "token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Contains(t, string(body), "resource_exhausted")
	assert.Zero(t, recorder.resultSize)
}
