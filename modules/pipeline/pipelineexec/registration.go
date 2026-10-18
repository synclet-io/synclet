package pipelineexec

import (
	"context"

	"github.com/go-pnp/go-pnp/logging"
)

// Registrar keeps the executor listed in the control plane's executor
// registry: each Execute sends one registration heartbeat.
type Registrar struct {
	backend ExecutorBackend
	info    ExecutorInfo
	logger  *logging.Logger
}

// NewRegistrar returns a Registrar that reports info through backend.
func NewRegistrar(backend ExecutorBackend, info ExecutorInfo, logger *logging.Logger) *Registrar {
	return &Registrar{backend: backend, info: info, logger: logger}
}

// Execute sends one registration heartbeat. A failure is logged and not
// returned: the next heartbeat retries it.
func (r *Registrar) Execute(ctx context.Context) error {
	if err := r.backend.ExecutorHeartbeat(ctx, r.info); err != nil {
		r.logger.WithError(err).Error(ctx, "executor registration heartbeat failed", "executor_id", r.info.ID)
	}

	return nil
}
