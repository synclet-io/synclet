package pipelinesync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncWorkerManager_RunJob(t *testing.T) {
	mgr := NewSyncWorkerManager(context.Background(), (*logging.Logger)(nil))

	executed := make(chan struct{})

	mgr.RunJob(func(ctx context.Context) {
		close(executed)
	})

	select {
	case <-executed:
		// Job executed in a separate goroutine.
	case <-time.After(2 * time.Second):
		t.Fatal("RunJob function was not executed within timeout")
	}

	// Clean shutdown.
	require.NoError(t, mgr.Shutdown(2*time.Second))
}

func TestSyncWorkerManager_Shutdown_WaitsForActiveJobs(t *testing.T) {
	mgr := NewSyncWorkerManager(context.Background(), (*logging.Logger)(nil))

	release := make(chan struct{})
	var completed atomic.Int32

	// Start 3 blocking jobs.
	for range 3 {
		mgr.RunJob(func(ctx context.Context) {
			<-release
			completed.Add(1)
		})
	}

	// Shutdown should block while jobs are running.
	shutdownDone := make(chan error, 1)

	go func() {
		shutdownDone <- mgr.Shutdown(5 * time.Second)
	}()

	// Give Shutdown a moment to start.
	time.Sleep(50 * time.Millisecond)

	select {
	case <-shutdownDone:
		t.Fatal("Shutdown returned before jobs completed")
	default:
		// Expected: Shutdown is still blocking.
	}

	// Release all jobs.
	close(release)

	// Shutdown should return successfully.
	select {
	case err := <-shutdownDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not return after jobs completed")
	}

	assert.Equal(t, int32(3), completed.Load())
}

func TestSyncWorkerManager_Shutdown_CancelsContext(t *testing.T) {
	mgr := NewSyncWorkerManager(context.Background(), (*logging.Logger)(nil))

	jobDone := make(chan struct{})

	mgr.RunJob(func(ctx context.Context) {
		<-ctx.Done()
		close(jobDone)
	})

	// Give RunJob goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	err := mgr.Shutdown(5 * time.Second)
	require.NoError(t, err)

	// Verify the job received context cancellation.
	select {
	case <-jobDone:
		// Context was cancelled, job exited.
	case <-time.After(2 * time.Second):
		t.Fatal("job did not receive context cancellation")
	}
}

func TestSyncWorkerManager_Shutdown_Timeout(t *testing.T) {
	mgr := NewSyncWorkerManager(context.Background(), (*logging.Logger)(nil))

	// Start a job that never returns (ignores context cancellation).
	mgr.RunJob(func(ctx context.Context) {
		select {} // Block forever.
	})

	// Give RunJob goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	err := mgr.Shutdown(100 * time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestSyncWorkerManager_Context_DerivedFromParent(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())

	mgr := NewSyncWorkerManager(parentCtx, (*logging.Logger)(nil))

	jobCtxCancelled := make(chan struct{})

	mgr.RunJob(func(ctx context.Context) {
		<-ctx.Done()
		close(jobCtxCancelled)
	})

	// Give RunJob goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	// Cancel the parent context.
	parentCancel()

	// Verify manager's context (and thus the job's context) is cancelled.
	select {
	case <-jobCtxCancelled:
		// Parent cancellation propagated.
	case <-time.After(2 * time.Second):
		t.Fatal("parent context cancellation did not propagate to job")
	}

	// Manager's context should be done.
	select {
	case <-mgr.Context().Done():
		// Expected.
	default:
		t.Fatal("manager context should be cancelled after parent cancel")
	}
}
