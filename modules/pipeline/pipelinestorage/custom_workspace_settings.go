package pipelinestorage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// WorkspaceSettingsWriter provides upsert capability for workspace settings.
type WorkspaceSettingsWriter struct {
	db *gorm.DB
}

// NewWorkspaceSettingsWriter creates a new WorkspaceSettingsWriter.
func NewWorkspaceSettingsWriter(db *gorm.DB) *WorkspaceSettingsWriter {
	return &WorkspaceSettingsWriter{db: db}
}

// Upsert creates or updates workspace settings using ON CONFLICT DO UPDATE.
func (s *WorkspaceSettingsWriter) Upsert(ctx context.Context, settings *pipelineservice.WorkspaceSettings) (*pipelineservice.WorkspaceSettings, error) {
	settings.UpdatedAt = time.Now()
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = settings.UpdatedAt
	}

	model, err := convertWorkspaceSettingsToDB(settings)
	if err != nil {
		return nil, fmt.Errorf("converting workspace settings to db model: %w", err)
	}

	result := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "workspace_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"max_jobs_per_workspace", "updated_at"}),
		}).
		Create(model)
	if result.Error != nil {
		return nil, fmt.Errorf("upserting workspace settings: %w", result.Error)
	}

	out, err := convertWorkspaceSettingsFromDB(model)
	if err != nil {
		return nil, fmt.Errorf("converting workspace settings from db model: %w", err)
	}

	return out, nil
}
