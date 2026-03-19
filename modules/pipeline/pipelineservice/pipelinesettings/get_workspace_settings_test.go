package pipelinesettings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesettings"
)

func TestGetWorkspaceSettings_ReturnsDefault_WhenNotFound(t *testing.T) {
	wsID := uuid.New()
	storage := &mockStorage{
		workspaceSettings: &mockWorkspaceSettingssStorage{
			firstErr: pipelineservice.ErrWorkspaceSettingsNotFound,
		},
	}

	uc := pipelinesettings.NewGetWorkspaceSettings(storage)
	result, err := uc.Execute(context.Background(), wsID)

	require.NoError(t, err)
	assert.Equal(t, wsID, result.WorkspaceID)
	assert.Equal(t, 0, result.MaxJobsPerWorkspace)
}

func TestGetWorkspaceSettings_ReturnsStoredValue(t *testing.T) {
	wsID := uuid.New()
	storage := &mockStorage{
		workspaceSettings: &mockWorkspaceSettingssStorage{
			firstResult: &pipelineservice.WorkspaceSettings{
				WorkspaceID:         wsID,
				MaxJobsPerWorkspace: 50,
			},
		},
	}

	uc := pipelinesettings.NewGetWorkspaceSettings(storage)
	result, err := uc.Execute(context.Background(), wsID)

	require.NoError(t, err)
	assert.Equal(t, 50, result.MaxJobsPerWorkspace)
}

func TestGetWorkspaceSettings_PropagatesStorageError(t *testing.T) {
	storage := &mockStorage{
		workspaceSettings: &mockWorkspaceSettingssStorage{
			firstErr: fmt.Errorf("db connection lost"),
		},
	}

	uc := pipelinesettings.NewGetWorkspaceSettings(storage)
	_, err := uc.Execute(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "getting workspace settings")
}
