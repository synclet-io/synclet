package pipelinejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// RegisterExecutorParams describes an executor as its heartbeat reports it.
type RegisterExecutorParams struct {
	InstanceID       string
	Type             pipelineservice.ExecutorType
	Version          string
	Capacity         int
	Capabilities     pipelineservice.ExecutorCapabilities
	ResourceDefaults pipelineservice.ExecutorResourceDefaults
}

// RegisterExecutor records an executor heartbeat in the executor registry.
type RegisterExecutor struct {
	storage pipelineservice.Storage
	now     func() time.Time
}

// NewRegisterExecutor creates a new RegisterExecutor use case.
func NewRegisterExecutor(storage pipelineservice.Storage) *RegisterExecutor {
	return &RegisterExecutor{storage: storage, now: time.Now}
}

// Execute upserts the executor's registry row and stamps it as seen now.
func (uc *RegisterExecutor) Execute(ctx context.Context, params RegisterExecutorParams) error {
	now := uc.now()

	err := uc.storage.Executors().Upsert(ctx, &pipelineservice.Executor{
		ID:               uuid.New(),
		InstanceID:       params.InstanceID,
		Type:             params.Type,
		Version:          params.Version,
		Capacity:         params.Capacity,
		Capabilities:     params.Capabilities,
		ResourceDefaults: params.ResourceDefaults,
		FirstSeenAt:      now,
		LastSeenAt:       now,
	})
	if err != nil {
		return fmt.Errorf("registering executor %s: %w", params.InstanceID, err)
	}

	return nil
}
