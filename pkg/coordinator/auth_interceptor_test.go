package coordinator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexeck8s"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type fakeExecutorService struct {
	executorv1connect.UnimplementedExecutorServiceHandler

	cancelled  atomic.Bool
	heartbeats atomic.Int32

	mu      sync.Mutex
	secrets []string
}

func (s *fakeExecutorService) Heartbeat(_ context.Context, req *connect.Request[executorv1.HeartbeatRequest]) (*connect.Response[executorv1.HeartbeatResponse], error) {
	s.heartbeats.Add(1)
	s.mu.Lock()
	s.secrets = append(s.secrets, req.Header().Get(internalSecretHeader))
	s.mu.Unlock()

	return connect.NewResponse(&executorv1.HeartbeatResponse{Cancelled: s.cancelled.Load()}), nil
}

func startExecutorServer(t *testing.T, svc *fakeExecutorService, opts ...connect.HandlerOption) string {
	t.Helper()

	path, handler := executorv1connect.NewExecutorServiceHandler(svc, opts...)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv.URL
}

func writeToken(t *testing.T, token string) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, pipelineexeck8s.APITokenSecretKey), []byte(token+"\n"), 0o600))

	return dir
}

func TestReporter_SendsInternalSecretFromSecretsDir(t *testing.T) {
	svc := &fakeExecutorService{}
	url := startExecutorServer(t, svc, connect.WithInterceptors(connectutil.NewInternalSecretInterceptor("s3cret")))

	opts, err := clientOptions(writeToken(t, "s3cret"))
	require.NoError(t, err)

	reporter := NewReporter(url, "job-1", "conn-1", opts...)
	assert.False(t, reporter.SendHeartbeat(t.Context(), 0, 0))

	svc.mu.Lock()
	defer svc.mu.Unlock()

	assert.Equal(t, []string{"s3cret"}, svc.secrets)
}

func TestReporter_NoTokenFileSendsNoHeader(t *testing.T) {
	svc := &fakeExecutorService{}
	url := startExecutorServer(t, svc)

	opts, err := clientOptions(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, opts)

	NewReporter(url, "job-1", "conn-1", opts...).SendHeartbeat(t.Context(), 0, 0)

	svc.mu.Lock()
	defer svc.mu.Unlock()

	assert.Equal(t, []string{""}, svc.secrets)
}

func TestReporter_HeartbeatRejectedWithoutToken(t *testing.T) {
	svc := &fakeExecutorService{}
	svc.cancelled.Store(true)
	url := startExecutorServer(t, svc, connect.WithInterceptors(connectutil.NewInternalSecretInterceptor("s3cret")))

	assert.False(t, NewReporter(url, "job-1", "conn-1").SendHeartbeat(t.Context(), 0, 0))
	assert.Zero(t, svc.heartbeats.Load())
}

func TestHeartbeatUntilCancelled_CancelsOnServerSignal(t *testing.T) {
	prev := heartbeatInterval
	heartbeatInterval = 10 * time.Millisecond

	t.Cleanup(func() { heartbeatInterval = prev })

	svc := &fakeExecutorService{}
	url := startExecutorServer(t, svc)
	reporter := NewReporter(url, "job-1", "conn-1")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)

		heartbeatUntilCancelled(ctx, reporter, cancel)
	}()

	require.Eventually(t, func() bool { return svc.heartbeats.Load() >= 2 }, time.Second, 5*time.Millisecond)
	require.NoError(t, ctx.Err(), "run must keep going while the job is not cancelled")

	svc.cancelled.Store(true)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat loop did not stop after cancel")
	}

	assert.Error(t, ctx.Err(), "run context should be cancelled")
}
