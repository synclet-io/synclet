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

func TestBatchUpdateConnectors_EmptyIDs(t *testing.T) {
	store := newMockStorage()
	updateUC := pipelineconnectors.NewUpdateManagedConnector(store)
	batchUC := pipelineconnectors.NewBatchUpdateConnectors(updateUC, store)

	result, err := batchUC.Execute(context.Background(), pipelineconnectors.BatchUpdateConnectorsParams{
		WorkspaceID:  uuid.New(),
		ConnectorIDs: nil,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.UpdatedCount)
	assert.Empty(t, result.UpdatedConnectors)
}

func TestBatchUpdateConnectors_ExplicitIDs(t *testing.T) {
	store := newMockStorage()
	updateUC := pipelineconnectors.NewUpdateManagedConnector(store)
	batchUC := pipelineconnectors.NewBatchUpdateConnectors(updateUC, store)

	workspaceID := uuid.New()
	repoID := uuid.New()

	// Outdated connector -- will be requested for update.
	outdatedID := uuid.New()
	store.managedConnectors.connectors[outdatedID] = &pipelineservice.ManagedConnector{
		ID:            outdatedID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "0.1.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         `{"old":"spec"}`,
		RepositoryID: &repoID,
	}

	// Another outdated connector -- NOT requested for update.
	otherID := uuid.New()
	store.managedConnectors.connectors[otherID] = &pipelineservice.ManagedConnector{
		ID:            otherID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-mysql",
		DockerTag:     "0.1.0",
		Name:          "MySQL",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         `{"old":"spec"}`,
		RepositoryID: &repoID,
	}

	// Repo connectors.
	rcPostgres := uuid.New()
	store.repoConnectors.connectors[rcPostgres] = &pipelineservice.RepositoryConnector{
		ID:               rcPostgres,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "0.2.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Spec:             `{"new":"spec"}`,
	}
	rcMySQL := uuid.New()
	store.repoConnectors.connectors[rcMySQL] = &pipelineservice.RepositoryConnector{
		ID:               rcMySQL,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-mysql",
		DockerImageTag:   "0.2.0",
		Name:             "MySQL",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Spec:             `{"new":"spec"}`,
	}

	// Only request the outdated postgres connector.
	result, err := batchUC.Execute(context.Background(), pipelineconnectors.BatchUpdateConnectorsParams{
		WorkspaceID:  workspaceID,
		ConnectorIDs: []uuid.UUID{outdatedID},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.UpdatedCount)
	require.Len(t, result.UpdatedConnectors, 1)
	assert.Equal(t, outdatedID, result.UpdatedConnectors[0].ID)
	assert.Equal(t, "0.2.0", result.UpdatedConnectors[0].DockerTag)

	// Verify the other connector was NOT updated.
	assert.Equal(t, "0.1.0", store.managedConnectors.connectors[otherID].DockerTag)
}

func TestBatchUpdateConnectors_SkipsCustom(t *testing.T) {
	store := newMockStorage()
	updateUC := pipelineconnectors.NewUpdateManagedConnector(store)
	batchUC := pipelineconnectors.NewBatchUpdateConnectors(updateUC, store)

	workspaceID := uuid.New()

	// Custom connector (no repository link).
	customID := uuid.New()
	store.managedConnectors.connectors[customID] = &pipelineservice.ManagedConnector{
		ID:            customID,
		WorkspaceID:   workspaceID,
		DockerImage:   "custom/connector",
		DockerTag:     "latest",
		Name:          "Custom",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: nil, // No repository.
	}

	result, err := batchUC.Execute(context.Background(), pipelineconnectors.BatchUpdateConnectorsParams{
		WorkspaceID:  workspaceID,
		ConnectorIDs: []uuid.UUID{customID},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.UpdatedCount)
	assert.Empty(t, result.UpdatedConnectors)
}
