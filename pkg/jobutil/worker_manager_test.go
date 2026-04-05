package jobutil

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerManager_RunJob(t *testing.T) {
	mgr := NewWorkerManager((*logging.Logger)(nil))

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, mgr.Close(ctx))
}

func TestWorkerManager_Close_WaitsForActiveJobs(t *testing.T) {
	mgr := NewWorkerManager((*logging.Logger)(nil))

	release := make(chan struct{})
	var completed atomic.Int32

	// Start 3 blocking jobs.
	for range 3 {
		mgr.RunJob(func(ctx context.Context) {
			<-release
			completed.Add(1)
		})
	}

	// Close should block while jobs are running.
	closeDone := make(chan error, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		closeDone <- mgr.Close(ctx)
	}()

	// Give Close a moment to start.
	time.Sleep(50 * time.Millisecond)

	select {
	case <-closeDone:
		t.Fatal("Close returned before jobs completed")
	default:
		// Expected: Close is still blocking.
	}

	// Release all jobs.
	close(release)

	// Close should return successfully.
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after jobs completed")
	}

	assert.Equal(t, int32(3), completed.Load())
}

func TestWorkerManager_Close_CancelsContext(t *testing.T) {
	mgr := NewWorkerManager((*logging.Logger)(nil))

	jobDone := make(chan struct{})

	mgr.RunJob(func(ctx context.Context) {
		<-ctx.Done()
		close(jobDone)
	})

	// Give RunJob goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, mgr.Close(ctx))

	// Verify the job received context cancellation.
	select {
	case <-jobDone:
		// Context was cancelled, job exited.
	case <-time.After(2 * time.Second):
		t.Fatal("job did not receive context cancellation")
	}
}

func TestWorkerManager_Close_Timeout(t *testing.T) {
	mgr := NewWorkerManager((*logging.Logger)(nil))

	// Start a job that never returns (ignores context cancellation).
	mgr.RunJob(func(ctx context.Context) {
		select {} // Block forever.
	})

	// Give RunJob goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := mgr.Close(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
