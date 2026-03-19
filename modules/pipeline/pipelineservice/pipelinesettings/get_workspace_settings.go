package pipelinesettings

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetWorkspaceSettings retrieves workspace settings, returning defaults when no row exists.
type GetWorkspaceSettings struct {
	storage pipelineservice.Storage
}

// NewGetWorkspaceSettings creates a new GetWorkspaceSettings use case.
func NewGetWorkspaceSettings(storage pipelineservice.Storage) *GetWorkspaceSettings {
	return &GetWorkspaceSettings{storage: storage}
}

// Execute returns workspace settings for the given workspace ID.
// Returns default settings (MaxJobsPerWorkspace=0, meaning unlimited) when no row exists.
func (uc *GetWorkspaceSettings) Execute(ctx context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error) {
	settings, err := uc.storage.WorkspaceSettingss().First(ctx, &pipelineservice.WorkspaceSettingsFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		var notFound pipelineservice.NotFoundError
		if errors.As(err, &notFound) {
			return &pipelineservice.WorkspaceSettings{
				WorkspaceID:         workspaceID,
				MaxJobsPerWorkspace: 0,
			}, nil
		}

		return nil, fmt.Errorf("getting workspace settings: %w", err)
	}

	return settings, nil
}
