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

func TestSetConnectorStarred(t *testing.T) {
	t.Run("star stores the flag and records a star event", func(t *testing.T) {
		store := newConnectorStorage()
		workspaceID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		audit := &recordingAudit{}

		got, err := pipelineconnectors.NewSetConnectorStarred(store, audit).Execute(context.Background(), pipelineconnectors.SetConnectorStarredParams{
			ConnectorID: mc.ID,
			WorkspaceID: workspaceID,
			Starred:     true,
		})
		require.NoError(t, err)
		assert.True(t, got.Starred)
		assert.True(t, store.managedConnectors.connectors[mc.ID].Starred)

		require.Len(t, audit.events, 1)
		event := audit.events[0]
		assert.Equal(t, "star", event.Action)
		assert.Equal(t, "connector", event.ResourceType)
		assert.Equal(t, workspaceID, event.WorkspaceID)
		assert.Equal(t, mc.ID, event.ResourceID)
		assert.Equal(t, mc.Name, event.ResourceLabel)
		assert.Equal(t, false, auditField(event.Before, "starred"))
		assert.Equal(t, true, auditField(event.After, "starred"))
		assert.Equal(t, mc.DockerImage, auditField(event.After, "docker_image"))
	})

	t.Run("star keeps a spec written after the connector was loaded", func(t *testing.T) {
		store := newConnectorStorage()
		workspaceID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		store.managed.afterFirst = func() { store.managedConnectors.connectors[mc.ID].Spec = `{"v":2}` }

		_, err := pipelineconnectors.NewSetConnectorStarred(store, &recordingAudit{}).Execute(context.Background(), pipelineconnectors.SetConnectorStarredParams{
			ConnectorID: mc.ID,
			WorkspaceID: workspaceID,
			Starred:     true,
		})
		require.NoError(t, err)
		assert.True(t, store.managedConnectors.connectors[mc.ID].Starred)
		assert.Equal(t, `{"v":2}`, store.managedConnectors.connectors[mc.ID].Spec)
	})

	t.Run("unstar clears the flag and records an unstar event", func(t *testing.T) {
		store := newConnectorStorage()
		workspaceID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeDestination)
		mc.Starred = true
		audit := &recordingAudit{}

		got, err := pipelineconnectors.NewSetConnectorStarred(store, audit).Execute(context.Background(), pipelineconnectors.SetConnectorStarredParams{
			ConnectorID: mc.ID,
			WorkspaceID: workspaceID,
			Starred:     false,
		})
		require.NoError(t, err)
		assert.False(t, got.Starred)
		assert.False(t, store.managedConnectors.connectors[mc.ID].Starred)

		require.Len(t, audit.events, 1)
		assert.Equal(t, "unstar", audit.events[0].Action)
		assert.Equal(t, true, auditField(audit.events[0].Before, "starred"))
		assert.Equal(t, false, auditField(audit.events[0].After, "starred"))
	})

	t.Run("connector of another workspace is not found and stays unstarred", func(t *testing.T) {
		store := newConnectorStorage()
		mc := store.addConnector(uuid.New(), pipelineservice.ConnectorTypeSource)
		audit := &recordingAudit{}

		_, err := pipelineconnectors.NewSetConnectorStarred(store, audit).Execute(context.Background(), pipelineconnectors.SetConnectorStarredParams{
			ConnectorID: mc.ID,
			WorkspaceID: uuid.New(),
			Starred:     true,
		})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
		assert.False(t, store.managedConnectors.connectors[mc.ID].Starred)
		assert.Empty(t, audit.events)
	})

	t.Run("load error is returned without an audit event", func(t *testing.T) {
		store := newConnectorStorage()
		workspaceID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		store.managed.firstErr = errStorage
		audit := &recordingAudit{}

		_, err := pipelineconnectors.NewSetConnectorStarred(store, audit).Execute(context.Background(), pipelineconnectors.SetConnectorStarredParams{
			ConnectorID: mc.ID,
			WorkspaceID: workspaceID,
			Starred:     true,
		})
		require.ErrorIs(t, err, errStorage)
		assert.Empty(t, audit.events)
	})

	t.Run("update error is returned without an audit event", func(t *testing.T) {
		store := newConnectorStorage()
		workspaceID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		store.managed.updateErr = errStorage
		audit := &recordingAudit{}

		_, err := pipelineconnectors.NewSetConnectorStarred(store, audit).Execute(context.Background(), pipelineconnectors.SetConnectorStarredParams{
			ConnectorID: mc.ID,
			WorkspaceID: workspaceID,
			Starred:     true,
		})
		require.ErrorIs(t, err, errStorage)
		assert.False(t, store.managedConnectors.connectors[mc.ID].Starred)
		assert.Empty(t, audit.events)
	})
}
