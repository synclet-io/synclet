package pipelinerepositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// AddRepository creates a repository and immediately syncs it to validate the URL.
// If the initial sync fails, the repository is deleted (per D-04).
type AddRepository struct {
	storage  pipelineservice.Storage
	syncRepo *SyncRepository
	secrets  pipelineservice.SecretsProvider
}

// NewAddRepository creates a new AddRepository use case.
func NewAddRepository(storage pipelineservice.Storage, syncRepo *SyncRepository, secrets pipelineservice.SecretsProvider) *AddRepository {
	return &AddRepository{storage: storage, syncRepo: syncRepo, secrets: secrets}
}

// AddRepositoryParams holds parameters for creating a repository.
type AddRepositoryParams struct {
	WorkspaceID uuid.UUID
	Name        string
	URL         string
	AuthHeader  *string
}

// Execute creates a repository record and immediately syncs it.
func (uc *AddRepository) Execute(ctx context.Context, params AddRepositoryParams) (*pipelineservice.Repository, error) {
	now := time.Now()
	repo := &pipelineservice.Repository{
		ID:             uuid.New(),
		WorkspaceID:    params.WorkspaceID,
		Name:           params.Name,
		URL:            params.URL,
		Status:         pipelineservice.RepositoryStatusSyncing,
		ConnectorCount: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Encrypt auth header as a secret before storing.
	if params.AuthHeader != nil && *params.AuthHeader != "" {
		secretRef, err := uc.secrets.StoreSecret(ctx, "repository", repo.ID, *params.AuthHeader)
		if err != nil {
			return nil, fmt.Errorf("encrypting auth header: %w", err)
		}
		repo.AuthHeader = &secretRef
	}

	created, err := uc.storage.Repositorys().Create(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("creating repository: %w", err)
	}

	// Immediately sync to validate the URL and populate connectors.
	syncedRepo, err := uc.syncRepo.Execute(ctx, SyncRepositoryParams{RepositoryID: created.ID, WorkspaceID: params.WorkspaceID})
	if err != nil {
		// Sync failed: delete the repository since the URL is not valid.
		// Clean up the encrypted secret (best-effort).
		_ = uc.secrets.DeleteByOwner(ctx, "repository", created.ID)
		_ = uc.storage.Repositorys().Delete(ctx, &pipelineservice.RepositoryFilter{
			ID: filter.Equals(created.ID),
		})
		return nil, fmt.Errorf("initial sync failed: %w", err)
	}

	return syncedRepo, nil
}
