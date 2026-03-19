package pipelinerepositories

import (
	"context"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ValidateImageParams holds parameters for validating a connector image.
type ValidateImageParams struct {
	DockerRepository string
}

// ValidateImage checks whether a connector image exists in any configured repository.
type ValidateImage struct {
	storage pipelineservice.Storage
}

// NewValidateImage creates a new ValidateImage use case.
func NewValidateImage(storage pipelineservice.Storage) *ValidateImage {
	return &ValidateImage{storage: storage}
}

// Execute returns an error if the docker repository is not found in any configured repository.
func (uc *ValidateImage) Execute(ctx context.Context, params ValidateImageParams) error {
	connectors, err := uc.storage.RepositoryConnectors().Find(ctx, &pipelineservice.RepositoryConnectorFilter{
		DockerRepository: filter.Equals(params.DockerRepository),
	})
	if err != nil {
		return fmt.Errorf("checking connector image: %w", err)
	}

	if len(connectors) == 0 {
		return fmt.Errorf("connector image %q is not found in any configured repository", params.DockerRepository)
	}

	return nil
}
