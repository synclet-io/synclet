package pipelinerepositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
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
// Returns the updated repository so callers don't need post-UC storage access.
func (uc *SyncRepository) Execute(ctx context.Context, params SyncRepositoryParams) (*pipelineservice.Repository, error) {
	// Load repository scoped to workspace.
	repoFilter := &pipelineservice.RepositoryFilter{
		ID:          filter.Equals(params.RepositoryID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}

	repo, err := uc.storage.Repositorys().First(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("loading repository: %w", err)
	}

	// Set status to Syncing.
	repo.Status = pipelineservice.RepositoryStatusSyncing

	repo.UpdatedAt = time.Now()
	if _, err := uc.storage.Repositorys().Update(ctx, repo); err != nil {
		return nil, fmt.Errorf("updating repository status to syncing: %w", err)
	}

	// Decrypt auth header if it's a secret reference (backward compatible with plaintext).
	authHeader := repo.AuthHeader
	if authHeader != nil && secretutil.IsSecretRef(*authHeader) {
		plaintext, err := uc.secrets.RetrieveSecret(ctx, *authHeader)
		if err != nil {
			return nil, fmt.Errorf("decrypting auth header: %w", err)
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
		_, _ = uc.storage.Repositorys().Update(ctx, repo)

		return nil, fmt.Errorf("fetching registry: %w", err)
	}

	// Replace all repository connectors in a transaction.
	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		// Delete existing connectors for this repository.
		if err := tx.RepositoryConnectors().Delete(ctx, &pipelineservice.RepositoryConnectorFilter{
			RepositoryID: filter.Equals(params.RepositoryID),
		}); err != nil {
			return fmt.Errorf("deleting old connectors: %w", err)
		}

		// Insert new connectors.
		for _, connData := range connectors {
			spec := connData.Spec
			if spec == "" {
				spec = "{}"
			}

			metadata := connData.Metadata
			if metadata == "" {
				metadata = "{}"
			}

			repoConnector := &pipelineservice.RepositoryConnector{
				ID:               uuid.New(),
				RepositoryID:     params.RepositoryID,
				DockerRepository: connData.DockerRepository,
				DockerImageTag:   connData.DockerImageTag,
				Name:             connData.Name,
				ConnectorType:    parseConnectorType(connData.ConnectorType),
				DocumentationURL: connData.DocumentationURL,
				ReleaseStage:     parseReleaseStage(connData.ReleaseStage),
				IconURL:          connData.IconURL,
				Spec:             spec,
				SupportLevel:     parseSupportLevel(connData.SupportLevel),
				License:          connData.License,
				SourceType:       parseSourceType(connData.SourceType),
				Metadata:         metadata,
			}
			if _, err := tx.RepositoryConnectors().Create(ctx, repoConnector); err != nil {
				return fmt.Errorf("creating connector %q: %w", connData.Name, err)
			}
		}

		// Update repository with new stats.
		now := time.Now()
		repo.Status = pipelineservice.RepositoryStatusSynced
		repo.LastSyncedAt = &now
		repo.ConnectorCount = len(connectors)
		repo.LastError = nil

		repo.UpdatedAt = now
		if _, err := tx.Repositorys().Update(ctx, repo); err != nil {
			return fmt.Errorf("updating repository: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("sync transaction: %w", err)
	}

	// Auto-create managed connectors for registry connectors (D-04).
	// Done outside the sync transaction to avoid bloat (Pitfall 2).
	if err := uc.autoCreateManagedConnectors(ctx, params.WorkspaceID, params.RepositoryID, connectors); err != nil {
		// Log warning but don't fail the sync -- the repo connectors are already saved.
		// Managed connector creation is best-effort during sync.
		uc.logger.WithError(err).WithFields(map[string]any{"workspace_id": params.WorkspaceID, "repository_id": params.RepositoryID}).Warn(ctx, "failed to auto-create managed connectors during sync")
	}

	return repo, nil
}

// autoCreateManagedConnectors creates managed connectors for all registry connectors
// that don't already exist in the workspace. Deduplicates by docker_image + workspace_id + repository_id.
func (uc *SyncRepository) autoCreateManagedConnectors(ctx context.Context, workspaceID, repositoryID uuid.UUID, connectors []ConnectorData) error {
	for _, connData := range connectors {
		// Check if managed connector already exists for this docker image in this workspace+repo (Pitfall 3).
		_, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
			WorkspaceID:  filter.Equals(workspaceID),
			DockerImage:  filter.Equals(connData.DockerRepository),
			RepositoryID: filter.Equals(&repositoryID),
		})
		if err == nil {
			// Already exists -- skip (D-07: never update existing managed connectors during sync).
			continue
		}

		// Only proceed to create if error is not-found. Other errors (DB connectivity) should be returned.
		var nfe pipelineservice.NotFoundError
		if !errors.As(err, &nfe) {
			return fmt.Errorf("checking existing managed connector %q: %w", connData.Name, err)
		}

		// Create new managed connector with READY status (D-05: skip PENDING/PULLING).
		spec := connData.Spec
		if spec == "" {
			spec = "{}"
		}

		now := time.Now()

		connector := &pipelineservice.ManagedConnector{
			ID:            uuid.New(),
			WorkspaceID:   workspaceID,
			DockerImage:   connData.DockerRepository,
			DockerTag:     connData.DockerImageTag,
			Name:          connData.Name,
			ConnectorType: parseConnectorType(connData.ConnectorType),
			Spec:          spec,
			CreatedAt:     now,
			UpdatedAt:     now,
			RepositoryID:  &repositoryID,
		}
		if _, err := uc.storage.ManagedConnectors().Create(ctx, connector); err != nil {
			return fmt.Errorf("creating managed connector %q: %w", connData.Name, err)
		}
	}

	return nil
}

func parseConnectorType(s string) pipelineservice.ConnectorType {
	switch strings.ToLower(s) {
	case "source":
		return pipelineservice.ConnectorTypeSource
	case "destination":
		return pipelineservice.ConnectorTypeDestination
	default:
		return pipelineservice.ConnectorTypeSource
	}
}

func parseSupportLevel(s string) pipelineservice.SupportLevel {
	switch strings.ToLower(s) {
	case "community":
		return pipelineservice.SupportLevelCommunity
	case "certified":
		return pipelineservice.SupportLevelCertified
	default:
		return pipelineservice.SupportLevelUnknown
	}
}

func parseSourceType(s string) pipelineservice.SourceType {
	switch strings.ToLower(s) {
	case "api":
		return pipelineservice.SourceTypeAPI
	case "database":
		return pipelineservice.SourceTypeDatabase
	case "file":
		return pipelineservice.SourceTypeFile
	default:
		return pipelineservice.SourceTypeUnknown
	}
}

func parseReleaseStage(s string) pipelineservice.ReleaseStage {
	switch strings.ToLower(s) {
	case "generally_available":
		return pipelineservice.ReleaseStageGenerallyAvailable
	case "beta":
		return pipelineservice.ReleaseStageBeta
	case "alpha":
		return pipelineservice.ReleaseStageAlpha
	case "custom":
		return pipelineservice.ReleaseStageCustom
	default:
		return pipelineservice.ReleaseStageUnknown
	}
}
