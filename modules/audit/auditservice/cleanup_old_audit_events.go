package auditservice

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
)

// CleanupOldAuditEvents deletes audit events that fall outside each
// workspace's effective retention window.
type CleanupOldAuditEvents struct {
	storage          Storage
	retentionStorage RetentionStorage
	config           Config
	logger           *logging.Logger
	now              func() time.Time
}

func NewCleanupOldAuditEvents(
	storage Storage,
	retentionStorage RetentionStorage,
	config Config,
	logger *logging.Logger,
) *CleanupOldAuditEvents {
	return &CleanupOldAuditEvents{
		storage:          storage,
		retentionStorage: retentionStorage,
		config:           config,
		logger:           logger.Named("cleanup-old-audit-events"),
		now:              time.Now,
	}
}

// Execute runs cleanup for every workspace that has audit events. A workspace
// whose deletion fails is logged and skipped.
func (uc *CleanupOldAuditEvents) Execute(ctx context.Context) error {
	workspaceIDs, err := uc.retentionStorage.DistinctWorkspaceIDs(ctx)
	if err != nil {
		return fmt.Errorf("listing audit workspaces: %w", err)
	}

	settings, err := uc.storage.AuditRetentionSettings().Find(ctx, &AuditRetentionSettingFilter{})
	if err != nil {
		return fmt.Errorf("listing audit retention settings: %w", err)
	}

	settingByWorkspace := make(map[uuid.UUID]*AuditRetentionSetting, len(settings))
	for _, setting := range settings {
		settingByWorkspace[setting.WorkspaceID] = setting
	}

	now := uc.now()

	for _, workspaceID := range workspaceIDs {
		days := uc.config.EffectiveRetentionDays(settingByWorkspace[workspaceID])
		cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)

		deleted, err := uc.retentionStorage.DeleteOlderThan(ctx, workspaceID, cutoff)
		if err != nil {
			uc.logger.WithError(err).WithField("workspace_id", workspaceID.String()).Warn(ctx, "audit retention cleanup failed for workspace")

			continue
		}

		if deleted > 0 {
			uc.logger.WithFields(map[string]interface{}{
				"workspace_id":   workspaceID.String(),
				"deleted":        deleted,
				"retention_days": days,
			}).Info(ctx, "deleted expired audit events")
		}
	}

	return nil
}
