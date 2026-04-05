package pipelinerepositories

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// SyncRepository fetches the registry URL and replaces all repository connectors.
type SyncRepository struct {
	storage pipelineservice.Storage
	fetcher *RegistryFetcher
	secrets pipelineservice.SecretsProvider
	logger  *logging.Logger
}

// NewSyncRepository creates a new SyncRepository use case.
func NewSyncRepository(storage pipelineservice.Storage, fetcher *RegistryFetcher, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *SyncRepository {
	return &SyncRepository{storage: storage, fetcher: fetcher, secrets: secrets, logger: logger}
}

// SyncRepositoryParams holds parameters for syncing a repository.
type SyncRepositoryParams struct {
	RepositoryID uuid.UUID
	WorkspaceID  uuid.UUID
}

// Execute fetches connectors from the repository URL and replaces all stored connector entries.
// WorkspaceID is required; the repository lookup is unconditionally scoped to the workspace (IDOR protection).
// The entire operation runs in a transaction with FOR UPDATE on the repository row to prevent concurrent syncs.
// Returns the updated repository so callers don't need post-UC storage access.
func (uc *SyncRepository) Execute(ctx context.Context, params SyncRepositoryParams) (*pipelineservice.Repository, error) {
	var result *pipelineservice.Repository

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		// Load repository scoped to workspace with row lock.
		repo, err := tx.Repositories().First(ctx, &pipelineservice.RepositoryFilter{
			ID:          filter.Equals(params.RepositoryID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		}, dbutil.WithForUpdate())
		if err != nil {
			return fmt.Errorf("loading repository: %w", err)
		}

		// Decrypt auth header if it's a secret reference (backward compatible with plaintext).
		authHeader := repo.AuthHeader
		if authHeader != nil && secretutil.IsSecretRef(*authHeader) {
			plaintext, err := uc.secrets.RetrieveSecret(ctx, *authHeader)
			if err != nil {
				return fmt.Errorf("decrypting auth header: %w", err)
			}

			authHeader = &plaintext
		}

		// Fetch connectors from registry URL.
		connectors, err := uc.fetcher.Fetch(ctx, repo.URL, authHeader)
		if err != nil {
			// Mark as failed.
			errMsg := err.Error()
			repo.Status = pipelineservice.RepositoryStatusFailed
			repo.LastError = &errMsg
			repo.UpdatedAt = time.Now()

			if _, updateErr := tx.Repositories().Update(ctx, repo); updateErr != nil {
				return fmt.Errorf("updating repository status to failed: %w (fetch error: %w)", updateErr, err)
			}

			return fmt.Errorf("fetching registry: %w", err)
		}

		// Upsert connectors and remove stale ones.
		if err := uc.syncRepositoryConnectors(ctx, tx, params.RepositoryID, connectors); err != nil {
			return err
		}

		// Update repository with new stats.
		now := time.Now()
		repo.Status = pipelineservice.RepositoryStatusSynced
		repo.LastSyncedAt = &now
		repo.ConnectorCount = len(connectors)
		repo.LastError = nil
		repo.UpdatedAt = now

		if _, err := tx.Repositories().Update(ctx, repo); err != nil {
			return fmt.Errorf("updating repository: %w", err)
		}

		// Auto-create managed connectors for registry connectors.
		// Done inside the transaction so everything is atomic.
		if err := uc.autoCreateManagedConnectors(ctx, tx, params.WorkspaceID, params.RepositoryID, connectors); err != nil {
			uc.logger.WithError(err).Warn(ctx, "failed to auto-create managed connectors during sync",
				"workspace_id", params.WorkspaceID,
				"repository_id", params.RepositoryID)
		}

		result = repo

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *SyncRepository) syncRepositoryConnectors(ctx context.Context, tx pipelineservice.Storage, repositoryID uuid.UUID, connectors []ConnectorData) error {
	// Fetch existing connectors for this repository.
	existing, err := tx.RepositoryConnectors().Find(ctx, &pipelineservice.RepositoryConnectorFilter{
		RepositoryID: filter.Equals(repositoryID),
	})
	if err != nil {
		return fmt.Errorf("listing existing connectors: %w", err)
	}

	existingByImage := make(map[string]*pipelineservice.RepositoryConnector, len(existing))
	for _, rc := range existing {
		existingByImage[rc.DockerRepository] = rc
	}

	seen := make(map[string]struct{}, len(connectors))

	for _, connData := range connectors {
		seen[connData.DockerRepository] = struct{}{}

		spec := connData.Spec
		if spec == "" {
			spec = "{}"
		}

		if existing, ok := existingByImage[connData.DockerRepository]; ok {
			// Update existing connector.
			existing.DockerImageTag = connData.DockerImageTag
			existing.Name = connData.Name
			existing.ConnectorType = connData.ConnectorType
			existing.DocumentationURL = connData.DocumentationURL
			existing.ReleaseStage = connData.ReleaseStage
			existing.IconURL = connData.IconURL
			existing.Spec = spec
			existing.SupportLevel = connData.SupportLevel
			existing.License = connData.License
			existing.SourceType = connData.SourceType
			existing.Metadata = connData.Metadata

			if _, err := tx.RepositoryConnectors().Update(ctx, existing); err != nil {
				return fmt.Errorf("updating connector %q: %w", connData.Name, err)
			}
		} else {
			// Create new connector.
			repoConnector := &pipelineservice.RepositoryConnector{
				ID:               uuid.New(),
				RepositoryID:     repositoryID,
				DockerRepository: connData.DockerRepository,
				DockerImageTag:   connData.DockerImageTag,
				Name:             connData.Name,
				ConnectorType:    connData.ConnectorType,
				DocumentationURL: connData.DocumentationURL,
				ReleaseStage:     connData.ReleaseStage,
				IconURL:          connData.IconURL,
				Spec:             spec,
				SupportLevel:     connData.SupportLevel,
				License:          connData.License,
				SourceType:       connData.SourceType,
				Metadata:         connData.Metadata,
			}
			if _, err := tx.RepositoryConnectors().Create(ctx, repoConnector); err != nil {
				return fmt.Errorf("creating connector %q: %w", connData.Name, err)
			}
		}
	}

	// Delete connectors no longer in the registry.
	for image, rc := range existingByImage {
		if _, ok := seen[image]; !ok {
			if err := tx.RepositoryConnectors().Delete(ctx, &pipelineservice.RepositoryConnectorFilter{
				ID: filter.Equals(rc.ID),
			}); err != nil {
				return fmt.Errorf("deleting stale connector %q: %w", rc.Name, err)
			}
		}
	}

	return nil
}

// autoCreateManagedConnectors creates managed connectors for all registry connectors
// that don't already exist in the workspace. Deduplicates by docker_image + workspace_id + repository_id.
func (uc *SyncRepository) autoCreateManagedConnectors(ctx context.Context, tx pipelineservice.Storage, workspaceID, repositoryID uuid.UUID, connectors []ConnectorData) error {
	// Fetch all existing managed connectors for this repo in one query.
	existing, err := tx.ManagedConnectors().Find(ctx, &pipelineservice.ManagedConnectorFilter{
		WorkspaceID:  filter.Equals(workspaceID),
		RepositoryID: filter.Equals(&repositoryID),
	})
	if err != nil {
		return fmt.Errorf("listing existing managed connectors: %w", err)
	}

	existingImages := make(map[string]struct{}, len(existing))
	for _, mc := range existing {
		existingImages[mc.DockerImage] = struct{}{}
	}

	now := time.Now()

	for _, connData := range connectors {
		if _, exists := existingImages[connData.DockerRepository]; exists {
			continue
		}

		spec := connData.Spec
		if spec == "" {
			spec = "{}"
		}

		connector := &pipelineservice.ManagedConnector{
			ID:            uuid.New(),
			WorkspaceID:   workspaceID,
			DockerImage:   connData.DockerRepository,
			DockerTag:     connData.DockerImageTag,
			Name:          connData.Name,
			ConnectorType: connData.ConnectorType,
			Spec:          spec,
			CreatedAt:     now,
			UpdatedAt:     now,
			RepositoryID:  &repositoryID,
		}
		if _, err := tx.ManagedConnectors().Create(ctx, connector); err != nil {
			return fmt.Errorf("creating managed connector %q: %w", connData.Name, err)
		}
	}

	return nil
}
