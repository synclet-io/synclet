// Package pipelineworkspaces holds use cases that act on all pipeline data of
// one workspace at once.
package pipelineworkspaces

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

var activeJobStatuses = []pipelineservice.JobStatus{
	pipelineservice.JobStatusScheduled,
	pipelineservice.JobStatusStarting,
	pipelineservice.JobStatusRunning,
}

// PurgeWorkspace removes every pipeline record owned by a deleted workspace.
type PurgeWorkspace struct {
	storage pipelineservice.Storage
	stats   pipelineservice.StatsStorage
	secrets pipelineservice.SecretsProvider
	archive pipelineservice.LogArchive
}

// NewPurgeWorkspace creates a new PurgeWorkspace use case.
func NewPurgeWorkspace(
	storage pipelineservice.Storage,
	stats pipelineservice.StatsStorage,
	secrets pipelineservice.SecretsProvider,
	archive pipelineservice.LogArchive,
) *PurgeWorkspace {
	return &PurgeWorkspace{storage: storage, stats: stats, secrets: secrets, archive: archive}
}

// Execute cancels the workspace's active jobs, then deletes its archived job
// logs and its connections in one transaction that locks the jobs first
// (cascading to jobs, attempts, logs, catalogs, state and stream generations),
// sources, destinations, connector tasks, managed connectors, repositories
// (cascading to repository connectors), workspace settings and stats
// rollups. Secrets are deleted before the rows that own them, so a failed
// step leaves the owners in place and a rerun finishes the job. Running it again on a purged
// workspace is a no-op.
func (uc *PurgeWorkspace) Execute(ctx context.Context, workspaceID uuid.UUID) error {
	if err := uc.purgeConnections(ctx, workspaceID); err != nil {
		return err
	}

	if err := uc.purgeSources(ctx, workspaceID); err != nil {
		return err
	}

	if err := uc.purgeDestinations(ctx, workspaceID); err != nil {
		return err
	}

	if err := uc.purgeConnectorTasks(ctx, workspaceID); err != nil {
		return err
	}

	if err := uc.storage.ManagedConnectors().Delete(ctx, &pipelineservice.ManagedConnectorFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting managed connectors: %w", err)
	}

	if err := uc.purgeRepositories(ctx, workspaceID); err != nil {
		return err
	}

	if err := uc.storage.WorkspaceSettings().Delete(ctx, &pipelineservice.WorkspaceSettingsFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting workspace settings: %w", err)
	}

	if err := uc.stats.DeleteWorkspaceRollups(ctx, workspaceID); err != nil {
		return fmt.Errorf("deleting stats rollups: %w", err)
	}

	return nil
}

func (uc *PurgeWorkspace) purgeConnections(ctx context.Context, workspaceID uuid.UUID) error {
	connections, err := uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing connections: %w", err)
	}

	if len(connections) == 0 {
		return nil
	}

	connectionIDs := make([]uuid.UUID, len(connections))
	for i, conn := range connections {
		connectionIDs[i] = conn.ID
	}

	if err := uc.cancelActiveJobs(ctx, connectionIDs); err != nil {
		return err
	}

	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		keys, err := tx.JobLogSegments().LockWorkspaceArchiveKeys(ctx, workspaceID)
		if err != nil {
			return fmt.Errorf("locking jobs: %w", err)
		}

		if err := uc.archive.Delete(ctx, keys...); err != nil {
			return fmt.Errorf("deleting archived job logs: %w", err)
		}

		if err := tx.Connections().Delete(ctx, &pipelineservice.ConnectionFilter{
			WorkspaceID: filter.Equals(workspaceID),
		}); err != nil {
			return fmt.Errorf("deleting connections: %w", err)
		}

		return nil
	})
}

func (uc *PurgeWorkspace) cancelActiveJobs(ctx context.Context, connectionIDs []uuid.UUID) error {
	jobs, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		ConnectionID: filter.In(connectionIDs...),
		Status:       filter.In(activeJobStatuses...),
	})
	if err != nil {
		return fmt.Errorf("listing active jobs: %w", err)
	}

	now := time.Now()
	reason := "workspace deleted"

	for _, job := range jobs {
		job.Status = pipelineservice.JobStatusCancelled
		job.CompletedAt = &now
		job.FailureReason = &reason

		if _, err := uc.storage.Jobs().UpdateIfStatus(ctx, job, activeJobStatuses...); err != nil {
			return fmt.Errorf("cancelling job %s: %w", job.ID, err)
		}
	}

	return nil
}

func (uc *PurgeWorkspace) purgeSources(ctx context.Context, workspaceID uuid.UUID) error {
	sources, err := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing sources: %w", err)
	}

	for _, src := range sources {
		if err := uc.secrets.DeleteByOwner(ctx, "source", src.ID); err != nil {
			return fmt.Errorf("deleting secrets of source %s: %w", src.ID, err)
		}
	}

	if err := uc.storage.Sources().Delete(ctx, &pipelineservice.SourceFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting sources: %w", err)
	}

	return nil
}

func (uc *PurgeWorkspace) purgeDestinations(ctx context.Context, workspaceID uuid.UUID) error {
	destinations, err := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing destinations: %w", err)
	}

	for _, dest := range destinations {
		if err := uc.secrets.DeleteByOwner(ctx, "destination", dest.ID); err != nil {
			return fmt.Errorf("deleting secrets of destination %s: %w", dest.ID, err)
		}
	}

	if err := uc.storage.Destinations().Delete(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting destinations: %w", err)
	}

	return nil
}

func (uc *PurgeWorkspace) purgeConnectorTasks(ctx context.Context, workspaceID uuid.UUID) error {
	tasks, err := uc.storage.ConnectorTasks().Find(ctx, &pipelineservice.ConnectorTaskFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing connector tasks: %w", err)
	}

	for _, task := range tasks {
		payload, ok := task.Payload.(*pipelineservice.CheckPayload)
		if !ok || payload.Config == nil {
			continue
		}

		if err := uc.secrets.DeleteByOwner(ctx, pipelineservice.SecretOwnerConnectorTask, task.ID); err != nil {
			return fmt.Errorf("deleting secrets of connector task %s: %w", task.ID, err)
		}
	}

	if err := uc.storage.ConnectorTasks().Delete(ctx, &pipelineservice.ConnectorTaskFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting connector tasks: %w", err)
	}

	return nil
}

func (uc *PurgeWorkspace) purgeRepositories(ctx context.Context, workspaceID uuid.UUID) error {
	repositories, err := uc.storage.Repositories().Find(ctx, &pipelineservice.RepositoryFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	for _, repo := range repositories {
		if err := uc.secrets.DeleteByOwner(ctx, "repository", repo.ID); err != nil {
			return fmt.Errorf("deleting secrets of repository %s: %w", repo.ID, err)
		}
	}

	if err := uc.storage.Repositories().Delete(ctx, &pipelineservice.RepositoryFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting repositories: %w", err)
	}

	return nil
}
