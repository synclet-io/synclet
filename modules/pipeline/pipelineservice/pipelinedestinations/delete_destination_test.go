package pipelinedestinations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/pkg/auditutil"
)

func TestDeleteDestination(t *testing.T) {
	ws := uuid.New()

	newDestination := func(workspaceID uuid.UUID, config string) *pipelineservice.Destination {
		return &pipelineservice.Destination{
			ID:                 uuid.New(),
			WorkspaceID:        workspaceID,
			Name:               "bq",
			ManagedConnectorID: uuid.New(),
			Config:             config,
		}
	}

	t.Run("deletes the destination, its secrets and records an audit event", func(t *testing.T) {
		dest := newDestination(ws, "")
		store := newDestStore(dest)
		secrets := newFakeSecrets()
		pwRef := secrets.seed(dest.ID, "pl41nt3xt-pw")
		otherRef := secrets.seed(uuid.New(), "other")
		dest.Config = `{"host":"a","password":"` + pwRef + `"}`
		audit := &captureAudit{}

		uc := pipelinedestinations.NewDeleteDestination(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: dest.ID, WorkspaceID: ws})
		require.NoError(t, err)
		assert.NotContains(t, store.destinations.records, dest.ID)
		assert.Equal(t, 0, secrets.countForOwner(dest.ID))
		assert.Equal(t, []secretOwner{{ownerType: "destination", ownerID: dest.ID}}, secrets.deletedOwners)
		assert.Contains(t, secrets.values, otherRef)

		require.Len(t, audit.events, 1)

		event := audit.events[0]
		assert.Equal(t, "delete", event.Action)
		assert.Equal(t, "destination", event.ResourceType)
		assert.Equal(t, dest.ID, event.ResourceID)
		assert.Equal(t, "bq", event.ResourceLabel)
		assert.Nil(t, event.After)

		before := payloadMap(t, event.Before)
		assert.Equal(t, "bq", before["name"])
		assert.Equal(t, dest.ManagedConnectorID.String(), before["managed_connector_id"])

		cfg := payloadMap(t, before["config"])
		assert.Equal(t, "a", cfg["host"])
		assert.Equal(t, auditutil.RedactMask, cfg["password"])
		assert.NotContains(t, eventJSON(t, event), "pl41nt3xt-pw")
	})

	t.Run("returns not found for an unknown destination", func(t *testing.T) {
		store := newDestStore()
		audit := &captureAudit{}

		uc := pipelinedestinations.NewDeleteDestination(store, newFakeSecrets(), audit)
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: uuid.New(), WorkspaceID: ws})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
		assert.Empty(t, audit.events)
	})

	t.Run("refuses cross-workspace delete", func(t *testing.T) {
		dest := newDestination(uuid.New(), `{"host":"a"}`)
		store := newDestStore(dest)
		secrets := newFakeSecrets()
		secrets.seed(dest.ID, "pw")

		audit := &captureAudit{}

		uc := pipelinedestinations.NewDeleteDestination(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: dest.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
		assert.Contains(t, store.destinations.records, dest.ID)
		assert.Equal(t, 1, secrets.countForOwner(dest.ID))
		assert.Empty(t, audit.events)
	})

	t.Run("fails when secret cleanup fails and keeps the destination", func(t *testing.T) {
		dest := newDestination(ws, `{"host":"a"}`)
		store := newDestStore(dest)
		secrets := newFakeSecrets()
		secrets.deleteByOwnerErr = errors.New("vault down")
		audit := &captureAudit{}

		uc := pipelinedestinations.NewDeleteDestination(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: dest.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, secrets.deleteByOwnerErr)
		assert.Contains(t, store.destinations.records, dest.ID)
		assert.Empty(t, audit.events)
	})

	t.Run("fails when storage rejects the delete", func(t *testing.T) {
		dest := newDestination(ws, `{"host":"a"}`)
		store := newDestStore(dest)
		store.destinations.deleteErr = errors.New("db down")
		audit := &captureAudit{}

		uc := pipelinedestinations.NewDeleteDestination(store, newFakeSecrets(), audit)
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: dest.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, store.destinations.deleteErr)
		assert.Empty(t, audit.events)
	})

	t.Run("refuses while connections reference the destination and keeps its secrets", func(t *testing.T) {
		dest := newDestination(ws, `{"host":"a"}`)
		store := newDestStore(dest)
		store.connections.records = []*pipelineservice.Connection{
			{ID: uuid.New(), WorkspaceID: ws, Name: "orders", DestinationID: dest.ID},
			{ID: uuid.New(), WorkspaceID: ws, Name: "customers", DestinationID: dest.ID},
			{ID: uuid.New(), WorkspaceID: ws, Name: "unrelated", DestinationID: uuid.New()},
		}
		secrets := newFakeSecrets()
		secrets.seed(dest.ID, "pw")

		audit := &captureAudit{}

		uc := pipelinedestinations.NewDeleteDestination(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: dest.ID, WorkspaceID: ws})

		var inUse *pipelineservice.InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, []string{"customers", "orders"}, inUse.ConnectionNames)
		assert.Contains(t, err.Error(), "customers, orders")
		assert.Contains(t, store.destinations.records, dest.ID)
		assert.Equal(t, 1, secrets.countForOwner(dest.ID))
		assert.Empty(t, secrets.deletedOwners)
		assert.Empty(t, audit.events)
	})

	t.Run("fails when listing referencing connections fails", func(t *testing.T) {
		dest := newDestination(ws, `{"host":"a"}`)
		store := newDestStore(dest)
		store.connections.findErr = errors.New("db down")
		secrets := newFakeSecrets()
		secrets.seed(dest.ID, "pw")

		uc := pipelinedestinations.NewDeleteDestination(store, secrets, &captureAudit{})
		err := uc.Execute(context.Background(), pipelinedestinations.DeleteDestinationParams{ID: dest.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, store.connections.findErr)
		assert.Contains(t, store.destinations.records, dest.ID)
		assert.Equal(t, 1, secrets.countForOwner(dest.ID))
	})
}
