package pipelineconnections

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
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
	// Tags replaces the connection's tags when set, even to an empty list.
	Tags *[]string
}

// UpdateConnection updates an existing connection within a workspace.
type UpdateConnection struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewUpdateConnection creates a new UpdateConnection use case.
func NewUpdateConnection(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *UpdateConnection {
	return &UpdateConnection{storage: storage, audit: audit}
}

// Execute finds the connection by ID and workspace, applies updates, and
// persists them. The connection row stays locked from the read to the write,
// so a workflow that takes the connection as a step in the meantime is seen.
func (uc *UpdateConnection) Execute(ctx context.Context, params UpdateConnectionParams) (*pipelineservice.Connection, error) {
	var before map[string]any

	var updated *pipelineservice.Connection

	err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID:          filter.Equals(params.ID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		}, dbutil.WithForUpdate())
		if err != nil {
			return fmt.Errorf("getting connection: %w", err)
		}

		before = connectionToAuditPayload(conn)

		if err := applyConnectionUpdate(ctx, tx, conn, params); err != nil {
			return err
		}

		conn.UpdatedAt = time.Now()

		updated, err = tx.Connections().Update(ctx, conn)
		if err != nil {
			return fmt.Errorf("updating connection: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "connection",
		ResourceID:    updated.ID,
		ResourceLabel: updated.Name,
		Before:        before,
		After:         connectionToAuditPayload(updated),
	})

	return updated, nil
}

func applyConnectionUpdate(ctx context.Context, tx pipelineservice.Storage, conn *pipelineservice.Connection, params UpdateConnectionParams) error {
	if params.Name != nil {
		conn.Name = *params.Name
	}

	if params.Schedule != nil {
		schedule := emptyToNil(*params.Schedule)
		if err := validateSchedule(schedule); err != nil {
			return err
		}

		if schedule != nil {
			if err := rejectScheduleOfWorkflowStep(ctx, tx, conn.ID); err != nil {
				return err
			}
		}

		conn.Schedule = schedule
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
		conn.CustomNamespaceFormat = emptyToNil(*params.CustomNamespaceFormat)
	}

	if params.StreamPrefix != nil {
		conn.StreamPrefix = emptyToNil(*params.StreamPrefix)
	}

	if params.Tags != nil {
		tags, err := NormalizeTags(*params.Tags)
		if err != nil {
			return err
		}

		conn.Tags = tags
	}

	return nil
}

// rejectScheduleOfWorkflowStep returns a *ValidationError when the connection
// is a step of a workflow, which runs it in place of a schedule of its own.
func rejectScheduleOfWorkflowStep(ctx context.Context, tx pipelineservice.Storage, connectionID uuid.UUID) error {
	steps, err := tx.WorkflowSteps().Count(ctx, &pipelineservice.WorkflowStepFilter{
		ConnectionID: filter.Equals(connectionID),
	})
	if err != nil {
		return fmt.Errorf("checking workflow steps of connection: %w", err)
	}

	if steps > 0 {
		return &pipelineservice.ValidationError{Field: "schedule", Message: "the connection is run by a workflow and has no schedule of its own"}
	}

	return nil
}
