package pipelinesources_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/pkg/auditutil"
)

func TestDeleteSource(t *testing.T) {
	ws := uuid.New()

	newSource := func(workspaceID uuid.UUID, config string) *pipelineservice.Source {
		return &pipelineservice.Source{
			ID:                 uuid.New(),
			WorkspaceID:        workspaceID,
			Name:               "pg",
			ManagedConnectorID: uuid.New(),
			Config:             config,
		}
	}

	t.Run("deletes the source, its secrets and records an audit event", func(t *testing.T) {
		src := newSource(ws, "")
		store := newSrcStore(src)
		secrets := newFakeSecrets()
		pwRef := secrets.seed(src.ID, "pl41nt3xt-pw")
		otherRef := secrets.seed(uuid.New(), "other")
		src.Config = `{"host":"a","password":"` + pwRef + `"}`
		audit := &captureAudit{}

		uc := pipelinesources.NewDeleteSource(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: src.ID, WorkspaceID: ws})
		require.NoError(t, err)
		assert.NotContains(t, store.sources.records, src.ID)
		assert.Equal(t, 0, secrets.countForOwner(src.ID))
		assert.Equal(t, []secretOwner{{ownerType: "source", ownerID: src.ID}}, secrets.deletedOwners)
		assert.Contains(t, secrets.values, otherRef)

		require.Len(t, audit.events, 1)

		event := audit.events[0]
		assert.Equal(t, "delete", event.Action)
		assert.Equal(t, "source", event.ResourceType)
		assert.Equal(t, src.ID, event.ResourceID)
		assert.Equal(t, "pg", event.ResourceLabel)
		assert.Nil(t, event.After)

		before := payloadMap(t, event.Before)
		assert.Equal(t, "pg", before["name"])
		assert.Equal(t, src.ManagedConnectorID.String(), before["managed_connector_id"])

		cfg := payloadMap(t, before["config"])
		assert.Equal(t, "a", cfg["host"])
		assert.Equal(t, auditutil.RedactMask, cfg["password"])
		assert.NotContains(t, eventJSON(t, event), "pl41nt3xt-pw")
	})

	t.Run("returns not found for an unknown source", func(t *testing.T) {
		store := newSrcStore()
		audit := &captureAudit{}

		uc := pipelinesources.NewDeleteSource(store, newFakeSecrets(), audit)
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: uuid.New(), WorkspaceID: ws})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
		assert.Empty(t, audit.events)
	})

	t.Run("refuses cross-workspace delete", func(t *testing.T) {
		src := newSource(uuid.New(), `{"host":"a"}`)
		store := newSrcStore(src)
		secrets := newFakeSecrets()
		secrets.seed(src.ID, "pw")

		audit := &captureAudit{}

		uc := pipelinesources.NewDeleteSource(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: src.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
		assert.Contains(t, store.sources.records, src.ID)
		assert.Equal(t, 1, secrets.countForOwner(src.ID))
		assert.Empty(t, audit.events)
	})

	t.Run("fails when secret cleanup fails and keeps the source", func(t *testing.T) {
		src := newSource(ws, `{"host":"a"}`)
		store := newSrcStore(src)
		secrets := newFakeSecrets()
		secrets.deleteByOwnerErr = errors.New("vault down")
		audit := &captureAudit{}

		uc := pipelinesources.NewDeleteSource(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: src.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, secrets.deleteByOwnerErr)
		assert.Contains(t, store.sources.records, src.ID)
		assert.Empty(t, audit.events)
	})

	t.Run("fails when storage rejects the delete", func(t *testing.T) {
		src := newSource(ws, `{"host":"a"}`)
		store := newSrcStore(src)
		store.sources.deleteErr = errors.New("db down")
		audit := &captureAudit{}

		uc := pipelinesources.NewDeleteSource(store, newFakeSecrets(), audit)
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: src.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, store.sources.deleteErr)
		assert.Empty(t, audit.events)
	})

	t.Run("refuses while connections reference the source and keeps its secrets", func(t *testing.T) {
		src := newSource(ws, `{"host":"a"}`)
		store := newSrcStore(src)
		store.connections.records = []*pipelineservice.Connection{
			{ID: uuid.New(), WorkspaceID: ws, Name: "orders", SourceID: src.ID},
			{ID: uuid.New(), WorkspaceID: ws, Name: "customers", SourceID: src.ID},
			{ID: uuid.New(), WorkspaceID: ws, Name: "unrelated", SourceID: uuid.New()},
		}
		secrets := newFakeSecrets()
		secrets.seed(src.ID, "pw")

		audit := &captureAudit{}

		uc := pipelinesources.NewDeleteSource(store, secrets, audit)
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: src.ID, WorkspaceID: ws})

		var inUse *pipelineservice.InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, []string{"customers", "orders"}, inUse.ConnectionNames)
		assert.Contains(t, err.Error(), "customers, orders")
		assert.Contains(t, store.sources.records, src.ID)
		assert.Equal(t, 1, secrets.countForOwner(src.ID))
		assert.Empty(t, secrets.deletedOwners)
		assert.Empty(t, audit.events)
	})

	t.Run("fails when listing referencing connections fails", func(t *testing.T) {
		src := newSource(ws, `{"host":"a"}`)
		store := newSrcStore(src)
		store.connections.findErr = errors.New("db down")
		secrets := newFakeSecrets()
		secrets.seed(src.ID, "pw")

		uc := pipelinesources.NewDeleteSource(store, secrets, &captureAudit{})
		err := uc.Execute(context.Background(), pipelinesources.DeleteSourceParams{ID: src.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, store.connections.findErr)
		assert.Contains(t, store.sources.records, src.ID)
		assert.Equal(t, 1, secrets.countForOwner(src.ID))
	})
}
