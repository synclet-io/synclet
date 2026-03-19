package pipelinerepositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListRepositories returns all repositories for a workspace.
type ListRepositories struct {
	storage pipelineservice.Storage
}

// NewListRepositories creates a new ListRepositories use case.
func NewListRepositories(storage pipelineservice.Storage) *ListRepositories {
	return &ListRepositories{storage: storage}
}

// ListRepositoriesParams holds parameters for listing repositories.
type ListRepositoriesParams struct {
	WorkspaceID uuid.UUID
}

// Execute returns all repositories for the given workspace.
func (uc *ListRepositories) Execute(ctx context.Context, params ListRepositoriesParams) ([]*pipelineservice.Repository, error) {
	repos, err := uc.storage.Repositorys().Find(ctx, &pipelineservice.RepositoryFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	return repos, nil
}
