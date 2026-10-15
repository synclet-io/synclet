package pipelinedestinations_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/pkg/auditutil"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

func TestCreateDestination(t *testing.T) {
	ws := uuid.New()

	t.Run("stores spec-flagged secrets and records a redacted audit event", func(t *testing.T) {
		store := newDestStore()
		conn := store.addConnector(ws, specWithPassword)
		secrets := newFakeSecrets()
		audit := &captureAudit{}

		uc := pipelinedestinations.NewCreateDestination(store, secrets, audit, newLogger())
		created, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{"host":"db.local","password":"pl41nt3xt-pw"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, ws, created.WorkspaceID)
		assert.Equal(t, "bq", created.Name)
		assert.Equal(t, conn.ID, created.ManagedConnectorID)

		stored := configMap(t, created.Config)
		assert.Equal(t, "db.local", stored["host"])

		ref, _ := stored["password"].(string)
		assert.True(t, secretutil.IsSecretRef(ref))

		plaintext, err := secrets.RetrieveSecret(context.Background(), ref)
		require.NoError(t, err)
		assert.Equal(t, "pl41nt3xt-pw", plaintext)
		assert.Equal(t, 1, secrets.countForOwner(created.ID))
		assert.Contains(t, store.destinations.records, created.ID)

		require.Len(t, audit.events, 1)

		event := audit.events[0]
		assert.Equal(t, "create", event.Action)
		assert.Equal(t, "destination", event.ResourceType)
		assert.Equal(t, created.ID, event.ResourceID)
		assert.Equal(t, "bq", event.ResourceLabel)
		assert.Nil(t, event.Before)

		after := payloadMap(t, event.After)
		assert.Equal(t, "bq", after["name"])
		assert.Equal(t, conn.ID.String(), after["managed_connector_id"])

		cfg := payloadMap(t, after["config"])
		assert.Equal(t, "db.local", cfg["host"])
		assert.Equal(t, auditutil.RedactMask, cfg["password"])
		assert.NotContains(t, eventJSON(t, event), "pl41nt3xt-pw")
	})

	t.Run("keeps config verbatim when the spec declares no secrets", func(t *testing.T) {
		store := newDestStore()
		conn := store.addConnector(ws, specWithoutSecrets)
		secrets := newFakeSecrets()

		uc := pipelinedestinations.NewCreateDestination(store, secrets, &captureAudit{}, newLogger())
		created, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{"host":"db.local"}`),
		})
		require.NoError(t, err)
		assert.JSONEq(t, `{"host":"db.local"}`, created.Config)
		assert.Empty(t, secrets.values)
	})

	t.Run("fails when the managed connector does not exist", func(t *testing.T) {
		store := newDestStore()
		audit := &captureAudit{}

		uc := pipelinedestinations.NewCreateDestination(store, newFakeSecrets(), audit, newLogger())
		_, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: uuid.New(),
			Config:             json.RawMessage(`{}`),
		})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
		assert.Empty(t, store.destinations.records)
		assert.Empty(t, audit.events)
	})

	t.Run("fails when the managed connector belongs to another workspace", func(t *testing.T) {
		store := newDestStore()
		conn := store.addConnector(uuid.New(), specWithoutSecrets)

		uc := pipelinedestinations.NewCreateDestination(store, newFakeSecrets(), &captureAudit{}, newLogger())
		_, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{}`),
		})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
		assert.Empty(t, store.destinations.records)
	})

	t.Run("fails when storing a secret fails", func(t *testing.T) {
		store := newDestStore()
		conn := store.addConnector(ws, specWithPassword)
		secrets := newFakeSecrets()
		secrets.storeErr = errors.New("vault down")
		audit := &captureAudit{}

		uc := pipelinedestinations.NewCreateDestination(store, secrets, audit, newLogger())
		_, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{"host":"db.local","password":"pw"}`),
		})
		require.ErrorIs(t, err, secrets.storeErr)
		assert.Empty(t, store.destinations.records)
		assert.Empty(t, audit.events)
	})

	t.Run("fails on invalid config JSON", func(t *testing.T) {
		store := newDestStore()
		conn := store.addConnector(ws, specWithPassword)

		uc := pipelinedestinations.NewCreateDestination(store, newFakeSecrets(), &captureAudit{}, newLogger())
		_, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{not json`),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing config JSON")
		assert.Empty(t, store.destinations.records)
	})

	t.Run("fails when storage rejects the destination", func(t *testing.T) {
		store := newDestStore()
		conn := store.addConnector(ws, specWithoutSecrets)
		store.destinations.createErr = errors.New("db down")
		audit := &captureAudit{}

		uc := pipelinedestinations.NewCreateDestination(store, newFakeSecrets(), audit, newLogger())
		_, err := uc.Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "bq",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{}`),
		})
		require.ErrorIs(t, err, store.destinations.createErr)
		assert.Empty(t, audit.events)
	})
}
