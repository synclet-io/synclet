package pipelinesources

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DeleteSource deletes a source from a workspace.
type DeleteSource struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
}

// NewDeleteSource creates a new DeleteSource use case.
func NewDeleteSource(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider) *DeleteSource {
	return &DeleteSource{storage: storage, secrets: secrets}
}

// DeleteSourceParams holds parameters for deleting a source.
type DeleteSourceParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute deletes a source.
func (uc *DeleteSource) Execute(ctx context.Context, params DeleteSourceParams) error {
	// Get source first to clean up secrets.
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting source for secret cleanup: %w", err)
	}

	// Clean up all associated secrets.
	if err := uc.secrets.DeleteByOwner(ctx, "source", src.ID); err != nil {
		return fmt.Errorf("deleting source secrets: %w", err)
	}

	err = uc.storage.Sources().Delete(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("deleting source: %w", err)
	}

	return nil
}
