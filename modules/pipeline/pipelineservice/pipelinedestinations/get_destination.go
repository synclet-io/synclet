package pipelinedestinations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetDestination retrieves a destination by ID within a workspace.
type GetDestination struct {
	storage pipelineservice.Storage
}

// NewGetDestination creates a new GetDestination use case.
func NewGetDestination(storage pipelineservice.Storage) *GetDestination {
	return &GetDestination{storage: storage}
}

// GetDestinationParams holds parameters for getting a destination.
type GetDestinationParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute retrieves a destination.
func (uc *GetDestination) Execute(ctx context.Context, params GetDestinationParams) (*pipelineservice.Destination, error) {
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting destination: %w", err)
	}

	return dest, nil
}
