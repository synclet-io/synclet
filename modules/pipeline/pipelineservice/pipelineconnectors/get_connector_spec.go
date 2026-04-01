package pipelineconnectors

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetConnectorSpec retrieves the cached spec for a ready connector.
type GetConnectorSpec struct {
	storage pipelineservice.Storage
}

// NewGetConnectorSpec creates a new GetConnectorSpec use case.
func NewGetConnectorSpec(storage pipelineservice.Storage) *GetConnectorSpec {
	return &GetConnectorSpec{storage: storage}
}

// GetConnectorSpecParams holds parameters for getting a connector spec.
type GetConnectorSpecParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// GetConnectorSpecResult holds the spec and optional external documentation URLs.
type GetConnectorSpecResult struct {
	Spec                      string
	ExternalDocumentationURLs []pipelineservice.ExternalDocumentationURL
}

// Execute retrieves the cached spec for a ready connector, scoped to workspace.
func (uc *GetConnectorSpec) Execute(ctx context.Context, params GetConnectorSpecParams) (*GetConnectorSpecResult, error) {
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting managed connector: %w", err)
	}

	if connector.Spec == "" || connector.Spec == "{}" {
		return nil, fmt.Errorf("connector %s has no cached spec", params.ID)
	}

	result := &GetConnectorSpecResult{Spec: connector.Spec}

	// Look up external documentation URLs from repository connector metadata.
	// Docs are non-critical, so errors are silently ignored.
	if connector.RepositoryID != nil {
		rc, err := uc.storage.RepositoryConnectors().First(ctx, &pipelineservice.RepositoryConnectorFilter{
			RepositoryID:     filter.Equals(*connector.RepositoryID),
			DockerRepository: filter.Equals(connector.DockerImage),
		})
		if err == nil {
			meta, mErr := pipelineservice.UnmarshalMetadata(rc.Metadata)
			if mErr == nil {
				result.ExternalDocumentationURLs = meta.ExternalDocumentationURLs
			}
		}
	}

	return result, nil
}
