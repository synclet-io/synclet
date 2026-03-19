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
	writer WorkspaceSettingsWriter
}

// NewUpdateWorkspaceSettings creates a new UpdateWorkspaceSettings use case.
func NewUpdateWorkspaceSettings(writer WorkspaceSettingsWriter) *UpdateWorkspaceSettings {
	return &UpdateWorkspaceSettings{writer: writer}
}

// Execute validates and upserts workspace settings.
func (uc *UpdateWorkspaceSettings) Execute(ctx context.Context, params UpdateWorkspaceSettingsParams) (*pipelineservice.WorkspaceSettings, error) {
	if params.MaxJobsPerWorkspace != nil && *params.MaxJobsPerWorkspace < 0 {
		return nil, &pipelineservice.ValidationError{Message: "max_jobs_per_workspace must be >= 0"}
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

	return result, nil
}
