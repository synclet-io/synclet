package jobutil

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-pnp/go-pnp/logging"
)

// WorkerManager coordinates background goroutine lifecycle. It owns a server-lifetime
// context and tracks active goroutines via WaitGroup for orderly shutdown.
// State transitions: idle(0) → running(1) → stopping(2).
type WorkerManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	state  atomic.Int32 // 0=idle, 1=running, 2=stopping
	logger *logging.Logger
}

const (
	stateIdle     int32 = 0
	stateRunning  int32 = 1
	stateStopping int32 = 2
)

// NewWorkerManager creates a new WorkerManager.
func NewWorkerManager(logger *logging.Logger) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel is stored and called in Close()

	m := &WorkerManager{
		ctx:    ctx,
		cancel: cancel,
		logger: logger.Named("worker-manager"),
	}
	m.state.Store(stateRunning)

	return m
}

// RunJob spawns a goroutine tracked by the WaitGroup. The function receives
// the manager's server-lifetime context. Returns false if the manager is
// already stopping/stopped.
func (m *WorkerManager) RunJob(job func(ctx context.Context)) bool {
	if m.state.Load() != stateRunning {
		return false
	}

	m.wg.Add(1)

	go func() {
		defer m.wg.Done()

		job(m.ctx)
	}()

	return true
}

// Context returns the manager's server-lifetime context.
func (m *WorkerManager) Context() context.Context {
	return m.ctx
}

// Close cancels the server-lifetime context (triggering all active jobs to stop)
// and waits for all tracked goroutines to finish, with a timeout.
func (m *WorkerManager) Close(ctx context.Context) error {
	if !m.state.CompareAndSwap(stateRunning, stateStopping) {
		return nil
	}

	m.cancel()

	done := make(chan struct{})

	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info(context.Background(), "all workers stopped gracefully")

		return nil
	case <-ctx.Done():
		return fmt.Errorf("stopping worker manager: %w", ctx.Err())
	}
}
