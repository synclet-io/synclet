package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// SettingsProvider retrieves workspace settings. Satisfied by *pipelinesettings.GetWorkspaceSettings.
type SettingsProvider interface {
	Execute(ctx context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error)
}

// CleanupOldJobs deletes excess terminal jobs per workspace based on retention settings.
type CleanupOldJobs struct {
	storage          pipelineservice.Storage
	retentionStorage pipelineservice.JobRetentionStorage
	settings         SettingsProvider
	logger           *logging.Logger
}

// NewCleanupOldJobs creates a new CleanupOldJobs use case.
func NewCleanupOldJobs(
	storage pipelineservice.Storage,
	retentionStorage pipelineservice.JobRetentionStorage,
	settings SettingsProvider,
	logger *logging.Logger,
) *CleanupOldJobs {
	return &CleanupOldJobs{
		storage:          storage,
		retentionStorage: retentionStorage,
		settings:         settings,
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

// ExecuteForWorkspace runs cleanup for a specific workspace.
func (uc *CleanupOldJobs) ExecuteForWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	ws, err := uc.settings.Execute(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("getting workspace settings: %w", err)
	}

	maxJobs := ws.MaxJobsPerWorkspace
	if maxJobs <= 0 {
		return nil // unlimited
	}

	deleted, err := uc.retentionStorage.DeleteOldestTerminalJobs(ctx, workspaceID, maxJobs)
	if err != nil {
		return fmt.Errorf("deleting old jobs: %w", err)
	}
	if deleted > 0 && uc.logger != nil {
		uc.logger.WithFields(map[string]interface{}{"workspace_id": workspaceID.String(), "deleted": deleted}).Info(ctx, "cleaned up old jobs")
	}
	return nil
}
