package pipelinerepositories

import (
	"context"
	"fmt"

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
	ConnectorImage string
}

// GetConnectorVersionsResult holds the result of the version lookup.
type GetConnectorVersionsResult struct {
	Versions      []string
	LatestVersion string
}

// Execute queries repository_connectors for the given docker image and returns available versions.
func (uc *GetConnectorVersions) Execute(ctx context.Context, params GetConnectorVersionsParams) (*GetConnectorVersionsResult, error) {
	connectors, err := uc.storage.RepositoryConnectors().Find(ctx, &pipelineservice.RepositoryConnectorFilter{
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
