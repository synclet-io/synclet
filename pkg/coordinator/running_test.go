package coordinator

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
)

type runningExecutorService struct {
	executorv1connect.UnimplementedExecutorServiceHandler

	mu      sync.Mutex
	running []string
}

func (s *runningExecutorService) MarkJobRunning(_ context.Context, req *connect.Request[executorv1.MarkJobRunningRequest]) (*connect.Response[executorv1.MarkJobRunningResponse], error) {
	s.mu.Lock()
	s.running = append(s.running, req.Msg.GetJobId())
	s.mu.Unlock()

	return connect.NewResponse(&executorv1.MarkJobRunningResponse{}), nil
}

func (s *runningExecutorService) ReportLog(context.Context, *connect.Request[executorv1.ReportLogRequest]) (*connect.Response[executorv1.ReportLogResponse], error) {
	return connect.NewResponse(&executorv1.ReportLogResponse{}), nil
}

func (s *runningExecutorService) runningJobs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.running...)
}

func startRunningServer(t *testing.T, svc *runningExecutorService) string {
	t.Helper()

	path, handler := executorv1connect.NewExecutorServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv.URL
}

func TestReporter_ReportRunningSendsJobID(t *testing.T) {
	svc := &runningExecutorService{}
	r := NewReporter(startRunningServer(t, svc), "job-1", "conn-1")
	r.Start(context.Background())

	r.ReportRunning(context.Background())
	r.Stop()

	assert.Equal(t, []string{"job-1"}, svc.runningJobs())
}

func TestRunPipeline_ReportsRunningOnceConnectorsConnect(t *testing.T) {
	svc := &runningExecutorService{}
	dir := t.TempDir()
	require.NoError(t, CreateFIFOs(dir))

	r := NewReporter(startRunningServer(t, svc), "job-1", "conn-1")
	r.Start(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		_, err := runPipeline(ctx, Config{DataDir: dir, SecretsDir: dir}, r)
		done <- err
	}()

	time.Sleep(200 * time.Millisecond)
	assert.Empty(t, svc.runningJobs(), "job reported running before its connectors connected")

	sourceOut, err := openFIFO(ctx, filepath.Join(dir, SourceStdoutFIFO), os.O_WRONLY, "", "")
	require.NoError(t, err)

	destIn, err := openFIFO(ctx, filepath.Join(dir, DestStdinFIFO), os.O_RDONLY, "", "")
	require.NoError(t, err)

	destOut, err := openFIFO(ctx, filepath.Join(dir, DestStdoutFIFO), os.O_WRONLY, "", "")
	require.NoError(t, err)

	require.Eventually(t, func() bool { return len(svc.runningJobs()) == 1 }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, []string{"job-1"}, svc.runningJobs())

	require.NoError(t, sourceOut.Close())

	_, _ = io.Copy(io.Discard, destIn)
	require.NoError(t, destIn.Close())
	require.NoError(t, destOut.Close())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("runPipeline did not return")
	}

	r.Stop()
}

func TestRunPipeline_FailsWhenConnectorExitsBeforeOpeningFIFO(t *testing.T) {
	tests := []struct {
		name         string
		exitCodeFile string
		openSource   bool
		wantErr      string
	}{
		{
			name:         "source",
			exitCodeFile: "source.exitcode",
			wantErr:      "source connector exited with code 2 before opening source-stdout.fifo",
		},
		{
			name:         "destination",
			exitCodeFile: "dest.exitcode",
			openSource:   true,
			wantErr:      "destination connector exited with code 2 before opening dest-stdin.fifo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &runningExecutorService{}
			dir := t.TempDir()
			require.NoError(t, CreateFIFOs(dir))

			r := NewReporter(startRunningServer(t, svc), "job-1", "conn-1")
			r.Start(context.Background())
			t.Cleanup(r.Stop)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if tt.openSource {
				go func() {
					sourceOut, err := openFIFO(ctx, filepath.Join(dir, SourceStdoutFIFO), os.O_WRONLY, "", "")
					if err == nil {
						t.Cleanup(func() { _ = sourceOut.Close() })
					}
				}()
			}

			require.NoError(t, os.WriteFile(filepath.Join(dir, tt.exitCodeFile), []byte("2\n"), 0o600))

			done := make(chan error, 1)

			go func() {
				_, err := runPipeline(ctx, Config{DataDir: dir, SecretsDir: dir}, r)
				done <- err
			}()

			select {
			case err := <-done:
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			case <-time.After(5 * time.Second):
				t.Fatal("runPipeline kept waiting for a connector that had already exited")
			}

			assert.Empty(t, svc.runningJobs())
		})
	}
}
