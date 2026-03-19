package pipelinesources

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetSource retrieves a source by ID within a workspace.
type GetSource struct {
	storage pipelineservice.Storage
}

// NewGetSource creates a new GetSource use case.
func NewGetSource(storage pipelineservice.Storage) *GetSource {
	return &GetSource{storage: storage}
}

// GetSourceParams holds parameters for getting a source.
type GetSourceParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute retrieves a source.
func (uc *GetSource) Execute(ctx context.Context, params GetSourceParams) (*pipelineservice.Source, error) {
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting source: %w", err)
	}

	return src, nil
}
