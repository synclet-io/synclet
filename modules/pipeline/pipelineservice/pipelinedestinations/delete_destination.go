package pipelinedestinations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DeleteDestination deletes a destination from a workspace.
type DeleteDestination struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
}

// NewDeleteDestination creates a new DeleteDestination use case.
func NewDeleteDestination(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider) *DeleteDestination {
	return &DeleteDestination{storage: storage, secrets: secrets}
}

// DeleteDestinationParams holds parameters for deleting a destination.
type DeleteDestinationParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute deletes a destination.
func (uc *DeleteDestination) Execute(ctx context.Context, params DeleteDestinationParams) error {
	// Get destination first to clean up secrets.
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting destination for secret cleanup: %w", err)
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

	return nil
}
