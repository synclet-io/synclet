package pipelinesources

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListSources returns all sources for a workspace.
type ListSources struct {
	storage pipelineservice.Storage
}

// NewListSources creates a new ListSources use case.
func NewListSources(storage pipelineservice.Storage) *ListSources {
	return &ListSources{storage: storage}
}

// ListSourcesParams holds parameters for listing sources.
type ListSourcesParams struct {
	WorkspaceID uuid.UUID
}

// Execute lists all sources in a workspace.
func (uc *ListSources) Execute(ctx context.Context, params ListSourcesParams) ([]*pipelineservice.Source, error) {
	sources, err := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	return sources, nil
}
