package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// SettingsProvider retrieves workspace settings. Satisfied by *pipelinesettings.GetWorkspaceSettings.
type SettingsProvider interface {
	Execute(ctx context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error)
}

const retentionBatchSize = 500

// CleanupOldJobs deletes excess terminal jobs per workspace based on retention settings.
type CleanupOldJobs struct {
	storage          pipelineservice.Storage
	retentionStorage pipelineservice.JobRetentionStorage
	settings         SettingsProvider
	archive          pipelineservice.LogArchive
	logger           *logging.Logger
}

// NewCleanupOldJobs creates a new CleanupOldJobs use case.
func NewCleanupOldJobs(
	storage pipelineservice.Storage,
	retentionStorage pipelineservice.JobRetentionStorage,
	settings SettingsProvider,
	archive pipelineservice.LogArchive,
	logger *logging.Logger,
) *CleanupOldJobs {
	return &CleanupOldJobs{
		storage:          storage,
		retentionStorage: retentionStorage,
		settings:         settings,
		archive:          archive,
		logger:           logger,
	}
}

// Execute runs cleanup for all workspaces that have connections.
func (uc *CleanupOldJobs) Execute(ctx context.Context) error {
	connections, err := uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{})
	if err != nil {
		return fmt.Errorf("listing connections: %w", err)
	}

	// Collect unique workspace IDs.
	seen := make(map[uuid.UUID]struct{})
	var workspaceIDs []uuid.UUID

	for _, conn := range connections {
		if _, ok := seen[conn.WorkspaceID]; !ok {
			seen[conn.WorkspaceID] = struct{}{}
			workspaceIDs = append(workspaceIDs, conn.WorkspaceID)
		}
	}

	for _, wsID := range workspaceIDs {
		if err := uc.ExecuteForWorkspace(ctx, wsID); err != nil {
			if uc.logger != nil {
				uc.logger.WithError(err).WithField("workspace_id", wsID.String()).Warn(ctx, "retention cleanup failed for workspace")
			}

			continue
		}
	}

	return nil
}

// ExecuteForWorkspace runs cleanup for a specific workspace. The archived
// logs of each batch of jobs are deleted before the jobs, so a failed object
// delete leaves the jobs in place for the next run.
func (uc *CleanupOldJobs) ExecuteForWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	ws, err := uc.settings.Execute(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("getting workspace settings: %w", err)
	}

	maxJobs := ws.MaxJobsPerWorkspace
	if maxJobs <= 0 {
		return nil // unlimited
	}

	var deleted int64

	for {
		ids, err := uc.retentionStorage.ListExcessTerminalJobIDs(ctx, workspaceID, maxJobs, retentionBatchSize)
		if err != nil {
			return fmt.Errorf("listing old jobs: %w", err)
		}

		if len(ids) == 0 {
			break
		}

		if err := uc.deleteArchivedLogs(ctx, ids); err != nil {
			return err
		}

		n, err := uc.retentionStorage.DeleteJobs(ctx, ids)
		if err != nil {
			return fmt.Errorf("deleting old jobs: %w", err)
		}

		deleted += n

		if len(ids) < retentionBatchSize {
			break
		}
	}

	if deleted > 0 && uc.logger != nil {
		uc.logger.WithFields(map[string]interface{}{"workspace_id": workspaceID.String(), "deleted": deleted}).Info(ctx, "cleaned up old jobs")
	}

	return nil
}

func (uc *CleanupOldJobs) deleteArchivedLogs(ctx context.Context, jobIDs []uuid.UUID) error {
	segments, err := uc.storage.JobLogSegments().Find(ctx, &pipelineservice.JobLogSegmentFilter{
		JobID:  filter.In(jobIDs...),
		Status: filter.Equals(pipelineservice.JobLogSegmentStatusArchived),
	})
	if err != nil {
		return fmt.Errorf("listing archived job logs: %w", err)
	}

	keys := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment.ArchiveKey != nil {
			keys = append(keys, *segment.ArchiveKey)
		}
	}

	if err := uc.archive.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("deleting archived job logs: %w", err)
	}

	return nil
}
