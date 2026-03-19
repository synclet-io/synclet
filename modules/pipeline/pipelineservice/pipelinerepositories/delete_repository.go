package pipelinerepositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// DeleteRepository removes a repository and nulls out managed connector foreign keys.
type DeleteRepository struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
}

// NewDeleteRepository creates a new DeleteRepository use case.
func NewDeleteRepository(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider) *DeleteRepository {
	return &DeleteRepository{storage: storage, secrets: secrets}
}

// DeleteRepositoryParams holds parameters for deleting a repository.
type DeleteRepositoryParams struct {
	RepositoryID uuid.UUID
	WorkspaceID  uuid.UUID
}

// DeleteRepositoryResult holds the result of deleting a repository.
type DeleteRepositoryResult struct {
	AffectedConnectors int
}

// Execute deletes a repository and returns the count of managed connectors that were disassociated.
func (uc *DeleteRepository) Execute(ctx context.Context, params DeleteRepositoryParams) (*DeleteRepositoryResult, error) {
	// Count managed connectors linked to this repository.
	affectedCount, err := uc.storage.ManagedConnectors().Count(ctx, &pipelineservice.ManagedConnectorFilter{
		RepositoryID: filter.Equals(&params.RepositoryID),
	})
	if err != nil {
		return nil, fmt.Errorf("counting affected connectors: %w", err)
	}

	// Null out repository_id on managed connectors.
	// The ON DELETE SET NULL FK constraint handles this automatically when we delete the repository,
	// but we count first so we can return the affected count.

	// Load the repository to check for an encrypted auth header secret.
	repoFilter := &pipelineservice.RepositoryFilter{
		ID:          filter.Equals(params.RepositoryID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}
	repo, err := uc.storage.Repositorys().First(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("loading repository: %w", err)
	}

	// Delete the associated auth header secret (best-effort).
	if repo.AuthHeader != nil && secretutil.IsSecretRef(*repo.AuthHeader) {
		_ = uc.secrets.DeleteSecret(ctx, *repo.AuthHeader)
	}

	// Delete the repository (CASCADE deletes repository_connectors automatically).
	// WorkspaceID filter unconditionally prevents cross-workspace deletion.
	if err := uc.storage.Repositorys().Delete(ctx, repoFilter); err != nil {
		return nil, fmt.Errorf("deleting repository: %w", err)
	}

	return &DeleteRepositoryResult{AffectedConnectors: affectedCount}, nil
}
