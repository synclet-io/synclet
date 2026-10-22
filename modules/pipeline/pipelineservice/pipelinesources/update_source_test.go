package pipelinesources_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/pkg/auditutil"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

type updateFixture struct {
	store   *srcStore
	conn    *pipelineservice.ManagedConnector
	src     *pipelineservice.Source
	secrets *fakeSecrets
	audit   *captureAudit
	uc      *pipelinesources.UpdateSource
}

func newUpdateFixture(ws uuid.UUID, spec, config string) *updateFixture {
	store := newSrcStore()
	conn := store.addConnector(ws, spec)
	src := &pipelineservice.Source{
		ID:                 uuid.New(),
		WorkspaceID:        ws,
		Name:               "pg",
		ManagedConnectorID: conn.ID,
		Config:             config,
	}
	store.sources.records[src.ID] = src

	secrets := newFakeSecrets()
	audit := &captureAudit{}

	return &updateFixture{
		store:   store,
		conn:    conn,
		src:     src,
		secrets: secrets,
		audit:   audit,
		uc:      pipelinesources.NewUpdateSource(store, secrets, audit, newLogger()),
	}
}

func (f *updateFixture) stored() *pipelineservice.Source {
	return f.store.sources.records[f.src.ID]
}

func rawJSON(s string) *json.RawMessage {
	raw := json.RawMessage(s)

	return &raw
}

func TestUpdateSource(t *testing.T) {
	ws := uuid.New()

	t.Run("updates name only and leaves config untouched", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, `{"host":"a","password":"$secret:ref"}`)

		updated, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
			WorkspaceID: ws,
			Name:        ptr("pg-renamed"),
		})
		require.NoError(t, err)
		assert.Equal(t, "pg-renamed", updated.Name)
		assert.JSONEq(t, `{"host":"a","password":"$secret:ref"}`, updated.Config)
		assert.Equal(t, "pg-renamed", fx.stored().Name)
		assert.Empty(t, fx.secrets.values)

		require.Len(t, fx.audit.events, 1)

		event := fx.audit.events[0]
		assert.Equal(t, "update", event.Action)
		assert.Equal(t, "source", event.ResourceType)
		assert.Equal(t, fx.src.ID, event.ResourceID)

		before := payloadMap(t, event.Before)
		after := payloadMap(t, event.After)
		assert.Equal(t, "pg", before["name"])
		assert.Equal(t, "pg-renamed", after["name"])
		assert.Equal(t, before["config"], after["config"])
		assert.Nil(t, before["runtime_config"])
		assert.Nil(t, after["runtime_config"])
	})

	t.Run("re-encrypts a changed secret and deletes the previous one", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, "")
		oldRef := fx.secrets.seed(fx.src.ID, "old-pw")
		fx.src.Config = `{"host":"a","password":"` + oldRef + `"}`

		updated, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
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
		assert.Equal(t, 1, fx.secrets.countForOwner(fx.src.ID))

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
		oldRef := fx.secrets.seed(fx.src.ID, "old-pw")
		fx.src.Config = `{"host":"a","password":"` + oldRef + `"}`

		updated, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
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

		updated, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            fx.src.ID,
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

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            fx.src.ID,
			WorkspaceID:   ws,
			RuntimeConfig: ptr(`{"cpu_limit":"lots"}`),
		})
		assertRuntimeConfigRefused(t, err, "invalid runtime config")
		assert.Nil(t, fx.stored().RuntimeConfig)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("rejects a runtime config that is not a JSON object", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            fx.src.ID,
			WorkspaceID:   ws,
			RuntimeConfig: ptr(`not json`),
		})
		assertRuntimeConfigRefused(t, err, "not a JSON object")
		assert.Nil(t, fx.stored().RuntimeConfig)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("rejects a runtime config with an unknown key", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            fx.src.ID,
			WorkspaceID:   ws,
			RuntimeConfig: ptr(`{"cpu_limt":"1"}`),
		})
		assertRuntimeConfigRefused(t, err, "cpu_limt")
		assert.Nil(t, fx.stored().RuntimeConfig)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("leaves secrets untouched when the runtime config is refused", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, "")
		oldRef := fx.secrets.seed(fx.src.ID, "old-pw")
		fx.src.Config = `{"host":"a","password":"` + oldRef + `"}`

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            fx.src.ID,
			WorkspaceID:   ws,
			Config:        rawJSON(`{"host":"b","password":"n3w-pw"}`),
			RuntimeConfig: ptr(`not json`),
		})
		assertRuntimeConfigRefused(t, err, "not a JSON object")
		assert.Empty(t, fx.secrets.deletedRefs)
		assert.Equal(t, "old-pw", fx.secrets.values[oldRef])
		assert.Equal(t, 1, fx.secrets.countForOwner(fx.src.ID))
		assert.Empty(t, fx.audit.events)
	})

	t.Run("clears runtime config overrides with an empty string", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)
		fx.src.RuntimeConfig = ptr(`{"cpu_limit":"1"}`)

		updated, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            fx.src.ID,
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

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
			WorkspaceID: ws,
			Name:        ptr("hijacked"),
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
		assert.Equal(t, "pg", fx.stored().Name)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("returns not found for an unknown source", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          uuid.New(),
			WorkspaceID: ws,
			Name:        ptr("ghost"),
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("fails when the connector spec is missing so secrets are never stored in plaintext", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPassword, `{"host":"a"}`)
		delete(fx.store.connectors.records, fx.conn.ID)

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
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

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
			WorkspaceID: ws,
			Config:      rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.ErrorIs(t, err, fx.secrets.storeErr)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("fails when storage rejects the update", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithoutSecrets, `{"host":"a"}`)
		fx.store.sources.updateErr = errors.New("db down")

		_, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
			WorkspaceID: ws,
			Name:        ptr("pg2"),
		})
		require.ErrorIs(t, err, fx.store.sources.updateErr)
		assert.Empty(t, fx.audit.events)
	})

	t.Run("audit event never contains plaintext secrets", func(t *testing.T) {
		fx := newUpdateFixture(ws, specWithPatOnly, "")
		oldPatRef := fx.secrets.seed(fx.src.ID, "old-pat-value")
		fx.src.Config = `{"host":"a","pat":"` + oldPatRef + `","password":"old-password-value","access_token":"old-token-value"}`

		updated, err := fx.uc.Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:          fx.src.ID,
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

func TestUpdateSourceInternal(t *testing.T) {
	t.Run("updates name and config without workspace scoping", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithPassword, `{"host":"a"}`)
		uc := pipelinesources.NewUpdateSourceInternal(fx.store, fx.secrets, newLogger())

		updated, err := uc.Execute(context.Background(), pipelinesources.UpdateSourceInternalParams{
			ID:     fx.src.ID,
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
		uc := pipelinesources.NewUpdateSourceInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinesources.UpdateSourceInternalParams{
			ID:     fx.src.ID,
			Config: rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.Error(t, err)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
		assert.Empty(t, fx.secrets.values)
	})

	t.Run("fails when storing a secret fails", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithPassword, `{"host":"a"}`)
		fx.secrets.storeErr = errors.New("vault down")
		uc := pipelinesources.NewUpdateSourceInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinesources.UpdateSourceInternalParams{
			ID:     fx.src.ID,
			Config: rawJSON(`{"host":"b","password":"pw"}`),
		})
		require.ErrorIs(t, err, fx.secrets.storeErr)
		assert.JSONEq(t, `{"host":"a"}`, fx.stored().Config)
	})

	t.Run("returns not found for an unknown source", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithoutSecrets, `{"host":"a"}`)
		uc := pipelinesources.NewUpdateSourceInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinesources.UpdateSourceInternalParams{
			ID:   uuid.New(),
			Name: ptr("ghost"),
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})

	t.Run("fails when storage rejects the update", func(t *testing.T) {
		fx := newUpdateFixture(uuid.New(), specWithoutSecrets, `{"host":"a"}`)
		fx.store.sources.updateErr = errors.New("db down")
		uc := pipelinesources.NewUpdateSourceInternal(fx.store, fx.secrets, newLogger())

		_, err := uc.Execute(context.Background(), pipelinesources.UpdateSourceInternalParams{
			ID:   fx.src.ID,
			Name: ptr("pg2"),
		})
		require.ErrorIs(t, err, fx.store.sources.updateErr)
	})
}

func assertRuntimeConfigRefused(t *testing.T, err error, want string) {
	t.Helper()

	var validationErr *pipelineservice.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "runtime_config", validationErr.Field)
	assert.Contains(t, validationErr.Message, want)
}
