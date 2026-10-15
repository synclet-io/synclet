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

type updateFixture struct {
	store   *destStore
	conn    *pipelineservice.ManagedConnector
	dest    *pipelineservice.Destination
	secrets *fakeSecrets
	audit   *captureAudit
	uc      *pipelinedestinations.UpdateDestination
}

func newUpdateFixture(ws uuid.UUID, spec, config string) *updateFixture {
	store := newDestStore()
	conn := store.addConnector(ws, spec)
	dest := &pipelineservice.Destination{
		ID:                 uuid.New(),
		WorkspaceID:        ws,
		Name:               "bq",
		ManagedConnectorID: conn.ID,
		Config:             config,
	}
	store.destinations.records[dest.ID] = dest

	secrets := newFakeSecrets()
	audit := &captureAudit{}

	return &updateFixture{
		store:   store,
		conn:    conn,
		dest:    dest,
		secrets: secrets,
		audit:   audit,
		uc:      pipelinedestinations.NewUpdateDestination(store, secrets, audit, newLogger()),
	}
}

func (f *updateFixture) stored() *pipelineservice.Destination {
	return f.store.destinations.records[f.dest.ID]
}

func rawJSON(s string) *json.RawMessage {
	raw := json.RawMessage(s)

	return &raw
}

func TestUpdateDestination(t *testing.T) {
	ws := uuid.New()

	t.Run("updates name only and leaves config untouched", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, `{"host":"a","password":"$secret:ref"}`)

		updated, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Name:        ptr("bq-renamed"),
		})
		require.NoError(t, err)
		assert.Equal(t, "bq-renamed", updated.Name)
		assert.JSONEq(t, `{"host":"a","password":"$secret:ref"}`, updated.Config)
		assert.Equal(t, "bq-renamed", fx.stored().Name)
		assert.Empty(t, fx.secrets.values)

		require.Len(t, fx.audit.events, 1)

		event := fx.audit.events[0]
		assert.Equal(t, "update", event.Action)
		assert.Equal(t, "destination", event.ResourceType)
		assert.Equal(t, fx.dest.ID, event.ResourceID)

		before := payloadMap(t, event.Before)
		after := payloadMap(t, event.After)
		assert.Equal(t, "bq", before["name"])
		assert.Equal(t, "bq-renamed", after["name"])
		assert.Equal(t, before["config"], after["config"])
		assert.Nil(t, before["runtime_config"])
		assert.Nil(t, after["runtime_config"])
	})

	t.Run("re-encrypts a changed secret and deletes the previous one", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, "")
		oldRef := fx.secrets.seed(fx.dest.ID, "old-pw")
		fx.dest.Config = `{"host":"a","password":"` + oldRef + `"}`

		updated, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Config:      rawJSON(`{"host":"b","password":"n3w-pw"}`),
		})
		require.NoError(t, err)

		stored := configMap(t, updated.Config)
		assert.Equal(t, "b", stored["host"])

		newRef, _ := stored["password"].(string)
		assert.True(t, secretutil.IsSecretRef(newRef))
		assert.NotEqual(t, oldRef, newRef)
		assert.Equal(t, "n3w-pw", fx.secrets.values[newRef])
		assert.Equal(t, []string{oldRef}, fx.secrets.deletedRefs)
		assert.Equal(t, 1, fx.secrets.countForOwner(fx.dest.ID))

		require.Len(t, fx.audit.events, 1)

		event := fx.audit.events[0]
		beforeCfg := payloadMap(t, payloadMap(t, event.Before)["config"])
		afterCfg := payloadMap(t, payloadMap(t, event.After)["config"])
		assert.Equal(t, "a", beforeCfg["host"])
		assert.Equal(t, "b", afterCfg["host"])
		assert.Equal(t, auditutil.RedactMask, beforeCfg["password"])
		assert.Equal(t, auditutil.RedactMask, afterCfg["password"])

		serialized := eventJSON(t, event)
		assert.NotContains(t, serialized, "old-pw")
		assert.NotContains(t, serialized, "n3w-pw")
	})

	t.Run("keeps the existing secret when the mask placeholder is sent", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, "")
		oldRef := fx.secrets.seed(fx.dest.ID, "old-pw")
		fx.dest.Config = `{"host":"a","password":"` + oldRef + `"}`

		updated, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Config:      rawJSON(`{"host":"b","password":"` + secretutil.SecretMask + `"}`),
		})
		require.NoError(t, err)

		stored := configMap(t, updated.Config)
		assert.Equal(t, "b", stored["host"])
		assert.Equal(t, oldRef, stored["password"])
		assert.Empty(t, fx.secrets.deletedRefs)
		assert.Equal(t, "old-pw", fx.secrets.values[oldRef])
	})

	t.Run("stores a valid runtime config", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)

		updated, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:            fx.dest.ID,
			WorkspaceID:   ws,
			RuntimeConfig: ptr(`{"cpu_limit":"500m","memory_limit":"256Mi"}`),
		})
		require.NoError(t, err)
		require.NotNil(t, updated.RuntimeConfig)
		assert.JSONEq(t, `{"cpu_limit":"500m","memory_limit":"256Mi"}`, *updated.RuntimeConfig)

		require.Len(t, fx.audit.events, 1)

		event := fx.audit.events[0]
		assert.Nil(t, payloadMap(t, event.Before)["runtime_config"])

		afterRuntime := payloadMap(t, payloadMap(t, event.After)["runtime_config"])
		assert.Equal(t, "500m", afterRuntime["cpu_limit"])
		assert.Equal(t, "256Mi", afterRuntime["memory_limit"])
	})

	t.Run("rejects an invalid runtime config", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)

		_, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:            fx.dest.ID,
			WorkspaceID:   ws,
			RuntimeConfig: ptr(`{"cpu_limit":"lots"}`),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid runtime config")
		assert.Nil(t, fx.stored().RuntimeConfig)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("clears runtime config overrides with an empty string", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)
		fx.dest.RuntimeConfig = ptr(`{"cpu_limit":"1"}`)

		updated, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:            fx.dest.ID,
			WorkspaceID:   ws,
			RuntimeConfig: ptr(""),
		})
		require.NoError(t, err)
		require.NotNil(t, updated.RuntimeConfig)
		assert.Empty(t, *updated.RuntimeConfig)

		require.Len(t, fx.audit.events, 1)

		event := fx.audit.events[0]
		beforeRuntime := payloadMap(t, payloadMap(t, event.Before)["runtime_config"])
		assert.Equal(t, "1", beforeRuntime["cpu_limit"])
		assert.Nil(t, payloadMap(t, event.After)["runtime_config"])
	})

	t.Run("refuses cross-workspace update", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithoutSecrets, `{"host":"a"}`)

		_, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Name:        ptr("hijacked"),
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
		assert.Equal(t, "bq", fx.stored().Name)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("returns not found for an unknown destination", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)

		_, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          uuid.New(),
			WorkspaceID: ws,
			Name:        ptr("ghost"),
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("fails when the connector spec is missing so secrets are never stored in plaintext", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, `{"host":"a"}`)
		delete(fx.store.connectors.records, fx.conn.ID)

		_, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Config:      rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.Error(t, err)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
		assert.Empty(t, fx.secrets.values)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("fails when storing a secret fails", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, `{"host":"a"}`)
		fx.secrets.storeErr = errors.New("vault down")

		_, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Config:      rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.ErrorIs(t, err, fx.secrets.storeErr)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("fails when storage rejects the update", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)
		fx.store.destinations.updateErr = errors.New("db down")

		_, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Name:        ptr("bq2"),
		})
		require.ErrorIs(t, err, fx.store.destinations.updateErr)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("audit event never contains plaintext secrets", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPatOnly, "")
		oldPatRef := fx.secrets.seed(fx.dest.ID, "old-pat-value")
		fx.dest.Config = `{"host":"a","pat":"` + oldPatRef + `","password":"old-password-value","access_token":"old-token-value"}`

		updated, err := fx.uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:          fx.dest.ID,
			WorkspaceID: ws,
			Config:      rawJSON(`{"host":"b","pat":"new-pat-value","password":"new-password-value","access_token":"new-token-value"}`),
		})
		require.NoError(t, err)

		stored := configMap(t, updated.Config)
		newPatRef, _ := stored["pat"].(string)
		assert.True(t, secretutil.IsSecretRef(newPatRef))
		assert.Equal(t, "new-pat-value", fx.secrets.values[newPatRef])

		require.Len(t, fx.audit.events, 1)

		event := fx.audit.events[0]
		serialized := eventJSON(t, event)

		for _, secret := range []string{
			"old-pat-value", "new-pat-value",
			"old-password-value", "new-password-value",
			"old-token-value", "new-token-value",
		} {
			assert.NotContains(t, serialized, secret)
		}

		beforeCfg := payloadMap(t, payloadMap(t, event.Before)["config"])
		afterCfg := payloadMap(t, payloadMap(t, event.After)["config"])
		assert.Equal(t, oldPatRef, beforeCfg["pat"])
		assert.Equal(t, newPatRef, afterCfg["pat"])
		assert.Equal(t, auditutil.RedactMask, afterCfg["password"])
		assert.Equal(t, auditutil.RedactMask, afterCfg["access_token"])
		assert.Equal(t, "b", afterCfg["host"])
	})
}

func TestUpdateDestinationInternal(t *testing.T) {
	t.Run("updates name and config without workspace scoping", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithPassword, `{"host":"a"}`)
		uc := pipelinedestinations.NewUpdateDestinationInternal(fx.store, fx.secrets, newLogger())

		updated, err := uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationInternalParams{
			ID:     fx.dest.ID,
			Name:   ptr("internal"),
			Config: rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, "internal", updated.Name)

		stored := configMap(t, updated.Config)
		assert.Equal(t, "b", stored["host"])

		ref, _ := stored["password"].(string)
		assert.True(t, secretutil.IsSecretRef(ref))
		assert.Equal(t, "pw", fx.secrets.values[ref])
		assert.Equal(t, "internal", fx.stored().Name)
	})

	t.Run("fails when the connector spec is missing so secrets are never stored in plaintext", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithPassword, `{"host":"a"}`)
		delete(fx.store.connectors.records, fx.conn.ID)
		uc := pipelinedestinations.NewUpdateDestinationInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationInternalParams{
			ID:     fx.dest.ID,
			Config: rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.Error(t, err)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
		assert.Empty(t, fx.secrets.values)
	})

	t.Run("fails when storing a secret fails", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithPassword, `{"host":"a"}`)
		fx.secrets.storeErr = errors.New("vault down")
		uc := pipelinedestinations.NewUpdateDestinationInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationInternalParams{
			ID:     fx.dest.ID,
			Config: rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.ErrorIs(t, err, fx.secrets.storeErr)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
	})

	t.Run("returns not found for an unknown destination", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithoutSecrets, `{"host":"a"}`)
		uc := pipelinedestinations.NewUpdateDestinationInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationInternalParams{
			ID:   uuid.New(),
			Name: ptr("ghost"),
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
	})

	t.Run("fails when storage rejects the update", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithoutSecrets, `{"host":"a"}`)
		fx.store.destinations.updateErr = errors.New("db down")
		uc := pipelinedestinations.NewUpdateDestinationInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinedestinations.UpdateDestinationInternalParams{
			ID:   fx.dest.ID,
			Name: ptr("bq2"),
		})
		require.ErrorIs(t, err, fx.store.destinations.updateErr)
	})
}
