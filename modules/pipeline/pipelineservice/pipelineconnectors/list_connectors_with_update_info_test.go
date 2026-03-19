package pipelineconnectors_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
)

func TestListConnectorsWithUpdateInfo_NoRepoID(t *testing.T) {
	store := newMockStorage()
	uc := pipelineconnectors.NewListConnectorsWithUpdateInfo(store)

	workspaceID := uuid.New()
	customID := uuid.New()
	store.managedConnectors.connectors[customID] = &pipelineservice.ManagedConnector{
		ID:            customID,
		WorkspaceID:   workspaceID,
		DockerImage:   "custom/connector",
		DockerTag:     "latest",
		Name:          "Custom",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: nil,
	}

	result, err := uc.Execute(context.Background(), pipelineconnectors.ListConnectorsWithUpdateInfoParams{
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, customID, result[0].Connector.ID)
	assert.Nil(t, result[0].UpdateInfo)
}

func TestListConnectorsWithUpdateInfo_UpToDate(t *testing.T) {
	store := newMockStorage()
	uc := pipelineconnectors.NewListConnectorsWithUpdateInfo(store)

	workspaceID := uuid.New()
	repoID := uuid.New()
	connID := uuid.New()

	store.managedConnectors.connectors[connID] = &pipelineservice.ManagedConnector{
		ID:            connID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "1.0.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: &repoID,
	}

	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
		ID:               rcID,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "1.0.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
	}

	result, err := uc.Execute(context.Background(), pipelineconnectors.ListConnectorsWithUpdateInfoParams{
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].UpdateInfo)
	assert.False(t, result[0].UpdateInfo.HasUpdate)
	assert.Equal(t, "1.0.0", result[0].UpdateInfo.AvailableVersion)
	assert.Nil(t, result[0].UpdateInfo.BreakingChanges)
}

func TestListConnectorsWithUpdateInfo_HasUpdate(t *testing.T) {
	store := newMockStorage()
	uc := pipelineconnectors.NewListConnectorsWithUpdateInfo(store)

	workspaceID := uuid.New()
	repoID := uuid.New()
	connID := uuid.New()

	store.managedConnectors.connectors[connID] = &pipelineservice.ManagedConnector{
		ID:            connID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "0.1.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: &repoID,
	}

	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
		ID:               rcID,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "0.2.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Metadata:         "{}",
	}

	result, err := uc.Execute(context.Background(), pipelineconnectors.ListConnectorsWithUpdateInfoParams{
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].UpdateInfo)
	assert.True(t, result[0].UpdateInfo.HasUpdate)
	assert.Equal(t, "0.2.0", result[0].UpdateInfo.AvailableVersion)
}

func TestListConnectorsWithUpdateInfo_WithBreakingChanges(t *testing.T) {
	store := newMockStorage()
	uc := pipelineconnectors.NewListConnectorsWithUpdateInfo(store)

	workspaceID := uuid.New()
	repoID := uuid.New()
	connID := uuid.New()

	store.managedConnectors.connectors[connID] = &pipelineservice.ManagedConnector{
		ID:            connID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "0.1.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: &repoID,
	}

	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
		ID:               rcID,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "1.0.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Metadata:         `{"breakingChanges":{"0.2.0":{"message":"Schema changed","migrationDocumentationUrl":"https://docs.example.com/migrate"},"0.5.0":{"message":"Auth changed","upgradeDeadline":"2025-01-01"}}}`,
	}

	result, err := uc.Execute(context.Background(), pipelineconnectors.ListConnectorsWithUpdateInfoParams{
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].UpdateInfo)
	assert.True(t, result[0].UpdateInfo.HasUpdate)
	assert.Equal(t, "1.0.0", result[0].UpdateInfo.AvailableVersion)
	require.Len(t, result[0].UpdateInfo.BreakingChanges, 2)
	assert.Equal(t, "0.2.0", result[0].UpdateInfo.BreakingChanges[0].Version)
	assert.Equal(t, "Schema changed", result[0].UpdateInfo.BreakingChanges[0].Message)
	assert.Equal(t, "0.5.0", result[0].UpdateInfo.BreakingChanges[1].Version)
	assert.Equal(t, "Auth changed", result[0].UpdateInfo.BreakingChanges[1].Message)
}

func TestGetConnectorWithUpdateInfo(t *testing.T) {
	store := newMockStorage()
	uc := pipelineconnectors.NewGetConnectorWithUpdateInfo(store)

	workspaceID := uuid.New()
	repoID := uuid.New()
	connID := uuid.New()

	store.managedConnectors.connectors[connID] = &pipelineservice.ManagedConnector{
		ID:            connID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "0.1.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: &repoID,
	}

	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
		ID:               rcID,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "0.2.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Metadata:         "{}",
	}

	result, err := uc.Execute(context.Background(), pipelineconnectors.GetConnectorWithUpdateInfoParams{
		ID:          connID,
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	assert.Equal(t, connID, result.Connector.ID)
	require.NotNil(t, result.UpdateInfo)
	assert.True(t, result.UpdateInfo.HasUpdate)
	assert.Equal(t, "0.2.0", result.UpdateInfo.AvailableVersion)
}
