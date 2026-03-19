package pipelinerepositories

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DefaultSyncInterval is the default interval between periodic repository syncs.
const DefaultSyncInterval = 1 * time.Hour

// RepositorySyncer periodically syncs all repositories in the background.
type RepositorySyncer struct {
	storage  pipelineservice.Storage
	syncRepo *SyncRepository
	interval time.Duration
	logger   *logging.Logger
}

// NewRepositorySyncer creates a new RepositorySyncer.
func NewRepositorySyncer(storage pipelineservice.Storage, syncRepo *SyncRepository, logger *logging.Logger) *RepositorySyncer {
	return &RepositorySyncer{
		storage:  storage,
		syncRepo: syncRepo,
		interval: DefaultSyncInterval,
		logger:   logger.Named("repository-syncer"),
	}
}

// Run starts the periodic sync loop. It blocks until the context is cancelled.
func (s *RepositorySyncer) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAll(ctx)
		}
	}
}

func (s *RepositorySyncer) syncAll(ctx context.Context) {
	repos, err := s.storage.Repositorys().Find(ctx, &pipelineservice.RepositoryFilter{})
	if err != nil {
		s.logger.WithError(err).Error(ctx, "failed to list repositories for sync")
		return
	}
	for _, repo := range repos {
		if repo.Status == pipelineservice.RepositoryStatusSyncing {
			continue // skip if already syncing
		}
		if _, err := s.syncRepo.Execute(ctx, SyncRepositoryParams{RepositoryID: repo.ID, WorkspaceID: repo.WorkspaceID}); err != nil {
			s.logger.WithError(err).Warn(ctx, "failed to sync repository",
				"repo_id", repo.ID.String(),
				"repo_name", repo.Name)
		}
	}
}
