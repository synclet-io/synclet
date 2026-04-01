package pipelineconnections

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// UpdateConnectionParams holds parameters for updating a connection.
type UpdateConnectionParams struct {
	ID                    uuid.UUID
	WorkspaceID           uuid.UUID
	Name                  *string
	Schedule              **string
	SchemaChangePolicy    *pipelineservice.SchemaChangePolicy
	MaxAttempts           *int
	NamespaceDefinition   *pipelineservice.NamespaceDefinition
	CustomNamespaceFormat **string
	StreamPrefix          **string
}

// UpdateConnection updates an existing connection within a workspace.
type UpdateConnection struct {
	storage pipelineservice.Storage
}

// NewUpdateConnection creates a new UpdateConnection use case.
func NewUpdateConnection(storage pipelineservice.Storage) *UpdateConnection {
	return &UpdateConnection{storage: storage}
}

// Execute finds the connection by ID and workspace, applies updates, and persists.
func (uc *UpdateConnection) Execute(ctx context.Context, params UpdateConnectionParams) (*pipelineservice.Connection, error) {
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	if params.Name != nil {
		conn.Name = *params.Name
	}

	if params.Schedule != nil {
		if *params.Schedule != nil && **params.Schedule != "" {
			if _, err := pipelineservice.CronParser.Parse(**params.Schedule); err != nil {
				return nil, fmt.Errorf("invalid cron expression %q: %w", **params.Schedule, err)
			}
		}

		conn.Schedule = *params.Schedule
		// Recompute next_scheduled_at when schedule changes
		pipelineservice.RecomputeNextScheduledAt(conn, time.Now())
	}

	if params.SchemaChangePolicy != nil {
		conn.SchemaChangePolicy = *params.SchemaChangePolicy
	}

	if params.MaxAttempts != nil {
		conn.MaxAttempts = *params.MaxAttempts
	}

	if params.NamespaceDefinition != nil {
		conn.NamespaceDefinition = *params.NamespaceDefinition
	}

	if params.CustomNamespaceFormat != nil {
		conn.CustomNamespaceFormat = *params.CustomNamespaceFormat
	}

	if params.StreamPrefix != nil {
		conn.StreamPrefix = *params.StreamPrefix
	}

	conn.UpdatedAt = time.Now()

	updated, err := uc.storage.Connections().Update(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("updating connection: %w", err)
	}

	return updated, nil
}
