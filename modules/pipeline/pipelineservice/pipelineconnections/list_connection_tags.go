package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListConnectionTagsParams holds parameters for listing a workspace's connection tags.
type ListConnectionTagsParams struct {
	WorkspaceID uuid.UUID
}

// ListConnectionTags lists the tags used by a workspace's connections.
type ListConnectionTags struct {
	storage pipelineservice.Storage
}

// NewListConnectionTags creates a new ListConnectionTags use case.
func NewListConnectionTags(storage pipelineservice.Storage) *ListConnectionTags {
	return &ListConnectionTags{storage: storage}
}

// Execute returns each tag of the workspace's connections with the number of
// connections carrying it, ordered by tag.
func (uc *ListConnectionTags) Execute(ctx context.Context, params ListConnectionTagsParams) ([]pipelineservice.TagCount, error) {
	tags, err := uc.storage.Connections().ListTagCounts(ctx, params.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing connection tags: %w", err)
	}

	if tags == nil {
		tags = []pipelineservice.TagCount{}
	}

	return tags, nil
}
