package pipelinedestinations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListDestinations returns all destinations for a workspace.
type ListDestinations struct {
	storage pipelineservice.Storage
}

// NewListDestinations creates a new ListDestinations use case.
func NewListDestinations(storage pipelineservice.Storage) *ListDestinations {
	return &ListDestinations{storage: storage}
}

// ListDestinationsParams holds parameters for listing destinations.
type ListDestinationsParams struct {
	WorkspaceID uuid.UUID
}

// Execute lists all destinations in a workspace.
func (uc *ListDestinations) Execute(ctx context.Context, params ListDestinationsParams) ([]*pipelineservice.Destination, error) {
	dests, err := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	return dests, nil
}
