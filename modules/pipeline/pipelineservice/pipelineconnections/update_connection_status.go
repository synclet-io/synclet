package pipelineconnections

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// UpdateConnectionStatusParams holds parameters for updating a connection's status.
type UpdateConnectionStatusParams struct {
	ID     uuid.UUID
	Status pipelineservice.ConnectionStatus
}

// UpdateConnectionStatus updates the status of a connection.
// This is an internal use case for the executor/scheduler (no workspace scoping).
type UpdateConnectionStatus struct {
	storage pipelineservice.Storage
}

// NewUpdateConnectionStatus creates a new UpdateConnectionStatus use case.
func NewUpdateConnectionStatus(storage pipelineservice.Storage) *UpdateConnectionStatus {
	return &UpdateConnectionStatus{storage: storage}
}

// Execute finds the connection by ID, updates its status, and persists.
func (uc *UpdateConnectionStatus) Execute(ctx context.Context, params UpdateConnectionStatusParams) (*pipelineservice.Connection, error) {
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	conn.Status = params.Status
	conn.UpdatedAt = time.Now()

	// Maintain next_scheduled_at based on new status
	pipelineservice.RecomputeNextScheduledAt(conn, time.Now())

	updated, err := uc.storage.Connections().Update(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("updating connection status: %w", err)
	}

	return updated, nil
}
