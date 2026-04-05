package pipelinerepositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// SyncAllRepositories syncs all repositories that are not currently syncing.
type SyncAllRepositories struct {
	storage  pipelineservice.Storage
	syncRepo *SyncRepository
}

// NewSyncAllRepositories creates a new SyncAllRepositories use case.
func NewSyncAllRepositories(storage pipelineservice.Storage, syncRepo *SyncRepository) *SyncAllRepositories {
	return &SyncAllRepositories{
		storage:  storage,
		syncRepo: syncRepo,
	}
}

// Execute lists all repositories and syncs each one that is not currently syncing.
func (uc *SyncAllRepositories) Execute(ctx context.Context) error {
	repos, err := uc.storage.Repositories().Find(ctx, &pipelineservice.RepositoryFilter{})
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	var errs []error

	for _, repo := range repos {
		if _, err := uc.syncRepo.Execute(ctx, SyncRepositoryParams{RepositoryID: repo.ID, WorkspaceID: repo.WorkspaceID}); err != nil {
			errs = append(errs, fmt.Errorf("repo %s (%s): %w", repo.Name, repo.ID, err))
		}
	}

	return errors.Join(errs...)
}
