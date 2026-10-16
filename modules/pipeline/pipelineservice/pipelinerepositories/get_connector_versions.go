package pipelinerepositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetConnectorVersions looks up the latest version of a connector by docker image.
type GetConnectorVersions struct {
	storage pipelineservice.Storage
}

// NewGetConnectorVersions creates a new GetConnectorVersions use case.
func NewGetConnectorVersions(storage pipelineservice.Storage) *GetConnectorVersions {
	return &GetConnectorVersions{storage: storage}
}

// GetConnectorVersionsParams holds parameters for getting connector versions.
type GetConnectorVersionsParams struct {
	WorkspaceID    uuid.UUID
	ConnectorImage string
}

// GetConnectorVersionsResult holds the result of the version lookup.
type GetConnectorVersionsResult struct {
	Versions      []string
	LatestVersion string
}

// Execute returns the versions of the given docker image listed by the workspace's repositories.
func (uc *GetConnectorVersions) Execute(ctx context.Context, params GetConnectorVersionsParams) (*GetConnectorVersionsResult, error) {
	repos, err := uc.storage.Repositories().Find(ctx, &pipelineservice.RepositoryFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing workspace repositories: %w", err)
	}

	if len(repos) == 0 {
		return &GetConnectorVersionsResult{}, nil
	}

	repoIDs := make([]uuid.UUID, 0, len(repos))
	for _, repo := range repos {
		repoIDs = append(repoIDs, repo.ID)
	}

	connectors, err := uc.storage.RepositoryConnectors().Find(ctx, &pipelineservice.RepositoryConnectorFilter{
		RepositoryID:     filter.In(repoIDs...),
		DockerRepository: filter.Equals(params.ConnectorImage),
	})
	if err != nil {
		return nil, fmt.Errorf("querying connector versions: %w", err)
	}

	if len(connectors) == 0 {
		return &GetConnectorVersionsResult{}, nil
	}

	// Return the latest version from the first matching record.
	latest := connectors[0].DockerImageTag

	return &GetConnectorVersionsResult{
		Versions:      []string{latest},
		LatestVersion: latest,
	}, nil
}
