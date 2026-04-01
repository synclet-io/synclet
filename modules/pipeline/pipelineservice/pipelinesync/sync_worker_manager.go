package pipelinesync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/logging"
)

// SyncWorkerManager coordinates sync worker lifecycle. It owns a server-lifetime
// context and tracks active sync goroutines via WaitGroup for orderly shutdown.
type SyncWorkerManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	logger *logging.Logger
}

// NewSyncWorkerManager creates a new SyncWorkerManager. The provided parentCtx
// should be the FX app context so FX signal handling propagates cancellation.
func NewSyncWorkerManager(parentCtx context.Context, logger *logging.Logger) *SyncWorkerManager {
	ctx, cancel := context.WithCancel(parentCtx) //nolint:gosec // cancel is stored and called in Stop()

	return &SyncWorkerManager{
		ctx:    ctx,
		cancel: cancel,
		logger: logger.Named("sync-worker-manager"),
	}
}

// RunJob spawns a goroutine tracked by the WaitGroup. The function receives
// the manager's server-lifetime context. wg.Done() runs AFTER fn returns,
// ensuring all deferred cleanup (container stops) completes before shutdown proceeds.
func (m *SyncWorkerManager) RunJob(job func(ctx context.Context)) {
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()

		job(m.ctx)
	}()
}

// Context returns the manager's server-lifetime context.
func (m *SyncWorkerManager) Context() context.Context {
	return m.ctx
}

// Shutdown cancels the server-lifetime context (triggering all active syncs to stop)
// and waits for all tracked goroutines to finish, with a timeout.
func (m *SyncWorkerManager) Shutdown(timeout time.Duration) error {
	m.cancel()

	done := make(chan struct{})

	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info(context.Background(), "all sync workers stopped gracefully")

		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timed out after %s with active syncs still running", timeout)
	}
}
