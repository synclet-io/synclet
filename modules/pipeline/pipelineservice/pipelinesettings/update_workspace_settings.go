package pipelinesettings

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// UpdateWorkspaceSettingsParams holds input for updating workspace settings.
type UpdateWorkspaceSettingsParams struct {
	WorkspaceID         uuid.UUID
	MaxJobsPerWorkspace *int
}

// WorkspaceSettingsWriter provides upsert capability for workspace settings.
// Satisfied by *pipelinestorage.WorkspaceSettingsWriter.
type WorkspaceSettingsWriter interface {
	Upsert(ctx context.Context, settings *pipelineservice.WorkspaceSettings) (*pipelineservice.WorkspaceSettings, error)
}

// UpdateWorkspaceSettings upserts workspace settings.
type UpdateWorkspaceSettings struct {
	reader *GetWorkspaceSettings
	writer WorkspaceSettingsWriter
	audit  pipelineservice.AuditRecorder
}

// NewUpdateWorkspaceSettings creates a new UpdateWorkspaceSettings use case.
func NewUpdateWorkspaceSettings(reader *GetWorkspaceSettings, writer WorkspaceSettingsWriter, audit pipelineservice.AuditRecorder) *UpdateWorkspaceSettings {
	return &UpdateWorkspaceSettings{reader: reader, writer: writer, audit: audit}
}

// Execute validates and upserts workspace settings.
func (uc *UpdateWorkspaceSettings) Execute(ctx context.Context, params UpdateWorkspaceSettingsParams) (*pipelineservice.WorkspaceSettings, error) {
	if params.MaxJobsPerWorkspace != nil && *params.MaxJobsPerWorkspace < 0 {
		return nil, &pipelineservice.ValidationError{Message: "max_jobs_per_workspace must be >= 0"}
	}

	current, err := uc.reader.Execute(ctx, params.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("loading current workspace settings: %w", err)
	}

	settings := &pipelineservice.WorkspaceSettings{
		WorkspaceID: params.WorkspaceID,
	}
	if params.MaxJobsPerWorkspace != nil {
		settings.MaxJobsPerWorkspace = *params.MaxJobsPerWorkspace
	}

	result, err := uc.writer.Upsert(ctx, settings)
	if err != nil {
		return nil, fmt.Errorf("upserting workspace settings: %w", err)
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:  params.WorkspaceID,
		Action:       "update",
		ResourceType: "workspace",
		ResourceID:   params.WorkspaceID,
		Before:       map[string]any{"max_jobs_per_workspace": current.MaxJobsPerWorkspace},
		After:        map[string]any{"max_jobs_per_workspace": result.MaxJobsPerWorkspace},
	})

	return result, nil
}
