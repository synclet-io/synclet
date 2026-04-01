package pipelinesettings_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesettings"
)

type mockWriter struct{ mock.Mock }

func (m *mockWriter) Upsert(ctx context.Context, settings *pipelineservice.WorkspaceSettings) (*pipelineservice.WorkspaceSettings, error) {
	args := m.Called(ctx, settings)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*pipelineservice.WorkspaceSettings), args.Error(1)
}

func TestUpdateWorkspaceSettings_Upserts(t *testing.T) {
	wsID := uuid.New()
	maxJobs := 100
	writer := &mockWriter{}
	writer.On("Upsert", mock.Anything, mock.MatchedBy(func(s *pipelineservice.WorkspaceSettings) bool {
		return s.WorkspaceID == wsID && s.MaxJobsPerWorkspace == 100
	})).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 100,
	}, nil)

	uc := pipelinesettings.NewUpdateWorkspaceSettings(writer)
	result, err := uc.Execute(context.Background(), pipelinesettings.UpdateWorkspaceSettingsParams{
		WorkspaceID: wsID, MaxJobsPerWorkspace: &maxJobs,
	})

	require.NoError(t, err)
	assert.Equal(t, 100, result.MaxJobsPerWorkspace)
	writer.AssertExpectations(t)
}

func TestUpdateWorkspaceSettings_RejectsNegative(t *testing.T) {
	writer := &mockWriter{}
	negative := -1
	uc := pipelinesettings.NewUpdateWorkspaceSettings(writer)

	_, err := uc.Execute(context.Background(), pipelinesettings.UpdateWorkspaceSettingsParams{
		WorkspaceID: uuid.New(), MaxJobsPerWorkspace: &negative,
	})

	require.Error(t, err)
	var validationErr *pipelineservice.ValidationError
	assert.ErrorAs(t, err, &validationErr)
}

func TestUpdateWorkspaceSettings_ZeroIsValid(t *testing.T) {
	wsID := uuid.New()
	zero := 0
	writer := &mockWriter{}
	writer.On("Upsert", mock.Anything, mock.Anything).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 0,
	}, nil)

	uc := pipelinesettings.NewUpdateWorkspaceSettings(writer)
	result, err := uc.Execute(context.Background(), pipelinesettings.UpdateWorkspaceSettingsParams{
		WorkspaceID: wsID, MaxJobsPerWorkspace: &zero,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.MaxJobsPerWorkspace)
}
