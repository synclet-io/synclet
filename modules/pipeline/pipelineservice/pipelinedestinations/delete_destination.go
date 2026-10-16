package pipelinedestinations

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DeleteDestination deletes a destination from a workspace.
type DeleteDestination struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	audit   pipelineservice.AuditRecorder
}

// NewDeleteDestination creates a new DeleteDestination use case.
func NewDeleteDestination(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, audit pipelineservice.AuditRecorder) *DeleteDestination {
	return &DeleteDestination{storage: storage, secrets: secrets, audit: audit}
}

// DeleteDestinationParams holds parameters for deleting a destination.
type DeleteDestinationParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute deletes a destination. It refuses with *pipelineservice.InUseError while
// any connection still references the destination.
func (uc *DeleteDestination) Execute(ctx context.Context, params DeleteDestinationParams) error {
	// Get destination first to clean up secrets.
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting destination for secret cleanup: %w", err)
	}

	if err := ensureDestinationUnused(ctx, uc.storage, dest); err != nil {
		return err
	}

	// Clean up all associated secrets.
	if err := uc.secrets.DeleteByOwner(ctx, "destination", dest.ID); err != nil {
		return fmt.Errorf("deleting destination secrets: %w", err)
	}

	err = uc.storage.Destinations().Delete(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("deleting destination: %w", err)
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "delete",
		ResourceType:  "destination",
		ResourceID:    dest.ID,
		ResourceLabel: dest.Name,
		Before: map[string]any{
			"name":                 dest.Name,
			"managed_connector_id": dest.ManagedConnectorID.String(),
			"config":               auditConfig(dest.Config),
		},
	})

	return nil
}

func ensureDestinationUnused(ctx context.Context, storage pipelineservice.Storage, dest *pipelineservice.Destination) error {
	connections, err := storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		WorkspaceID:   filter.Equals(dest.WorkspaceID),
		DestinationID: filter.Equals(dest.ID),
	})
	if err != nil {
		return fmt.Errorf("listing connections using destination: %w", err)
	}

	if len(connections) == 0 {
		return nil
	}

	names := make([]string, 0, len(connections))
	for _, conn := range connections {
		names = append(names, conn.Name)
	}

	slices.Sort(names)

	return &pipelineservice.InUseError{Resource: "destination " + dest.Name, ConnectionNames: names}
}
