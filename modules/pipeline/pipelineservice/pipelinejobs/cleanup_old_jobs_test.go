package pipelinejobs

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// mockSettingsProvider implements SettingsProvider (backed by *pipelinesettings.GetWorkspaceSettings).
type mockSettingsProvider struct {
	mock.Mock
}

func (m *mockSettingsProvider) Execute(ctx context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error) {
	args := m.Called(ctx, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*pipelineservice.WorkspaceSettings), args.Error(1)
}

// Verify interface compliance.
var _ SettingsProvider = (*mockSettingsProvider)(nil)

// mockRetentionStorage implements pipelineservice.JobRetentionStorage.
type mockRetentionStorage struct {
	mock.Mock
}

func (m *mockRetentionStorage) DeleteOldestTerminalJobs(ctx context.Context, workspaceID uuid.UUID, keepCount int) (int64, error) {
	args := m.Called(ctx, workspaceID, keepCount)
	return args.Get(0).(int64), args.Error(1)
}

// Verify interface compliance.
var _ pipelineservice.JobRetentionStorage = (*mockRetentionStorage)(nil)

func TestCleanupOldJobs_Unlimited(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	uc := NewCleanupOldJobs(nil, storage, settings, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 0,
	}, nil)

	err := uc.ExecuteForWorkspace(context.Background(), wsID)
	assert.NoError(t, err)

	// DeleteOldestTerminalJobs should NOT be called when maxJobs=0 (unlimited).
	storage.AssertNotCalled(t, "DeleteOldestTerminalJobs")
}

func TestCleanupOldJobs_DeletesExcess(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	uc := NewCleanupOldJobs(nil, storage, settings, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 50,
	}, nil)
	storage.On("DeleteOldestTerminalJobs", mock.Anything, wsID, 50).Return(int64(5), nil)

	err := uc.ExecuteForWorkspace(context.Background(), wsID)
	assert.NoError(t, err)

	storage.AssertCalled(t, "DeleteOldestTerminalJobs", mock.Anything, wsID, 50)
}

func TestCleanupOldJobs_StorageError(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	uc := NewCleanupOldJobs(nil, storage, settings, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 10,
	}, nil)
	storage.On("DeleteOldestTerminalJobs", mock.Anything, wsID, 10).Return(int64(0), fmt.Errorf("db error"))

	err := uc.ExecuteForWorkspace(context.Background(), wsID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deleting old jobs")
}

func TestCleanupOldJobs_SettingsError(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	uc := NewCleanupOldJobs(nil, storage, settings, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(nil, fmt.Errorf("workspace not found"))

	err := uc.ExecuteForWorkspace(context.Background(), wsID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "getting workspace settings")
}
