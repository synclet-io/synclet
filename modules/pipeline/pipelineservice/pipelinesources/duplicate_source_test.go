package pipelinesources_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

const specWithTwoSecrets = `{"connectionSpecification":{"type":"object","properties":{
	"host":{"type":"string"},
	"password":{"type":"string","airbyte_secret":true},
	"token":{"type":"string","airbyte_secret":true}}}}`

func TestDuplicateSourceSecrets(t *testing.T) {
	setup := func(t *testing.T, spec string) (*srcStore, *fakeSecrets, *pipelineservice.Source) {
		t.Helper()

		ws := uuid.New()
		store := newSrcStore()
		secrets := newFakeSecrets()
		connector := store.addConnector(ws, spec)
		srcID := uuid.New()
		src := &pipelineservice.Source{
			ID:                 srcID,
			WorkspaceID:        ws,
			Name:               "pg",
			ManagedConnectorID: connector.ID,
			Config:             `{"host":"db","password":"` + secrets.seed(srcID, "hunter2") + `","token":"` + secrets.seed(srcID, "t0k3n") + `"}`,
		}
		store.sources.records[src.ID] = src

		return store, secrets, src
	}

	duplicate := func(store *srcStore, secrets *fakeSecrets, src *pipelineservice.Source) (*pipelineservice.Source, error) {
		return pipelinesources.NewDuplicateSource(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinesources.DuplicateSourceParams{
			ID:            src.ID,
			WorkspaceID:   src.WorkspaceID,
			ExecutorGroup: "us",
		})
	}

	t.Run("a failed secret store leaves no secrets of the copy", func(t *testing.T) {
		store, secrets, src := setup(t, specWithTwoSecrets)
		secrets.storeErr = errors.New("vault down")
		secrets.failStoreAfter = 1

		_, err := duplicate(store, secrets, src)
		require.Error(t, err)
		assert.Len(t, secrets.owners, 2)
		assert.Equal(t, 2, secrets.countForOwner(src.ID))
		assert.Len(t, store.sources.records, 1)
	})

	t.Run("a reference owned by another source is not copied", func(t *testing.T) {
		store, secrets, src := setup(t, specWithTwoSecrets)
		src.Config = `{"host":"db","password":"` + secrets.seed(uuid.New(), "not-mine") + `"}`

		_, err := duplicate(store, secrets, src)
		require.Error(t, err)
		assert.Len(t, store.sources.records, 1)
		assert.Len(t, secrets.owners, 3)
	})

	t.Run("a failed create leaves no secrets of the copy", func(t *testing.T) {
		store, secrets, src := setup(t, specWithTwoSecrets)
		store.sources.createErr = errors.New("db down")

		_, err := duplicate(store, secrets, src)
		require.Error(t, err)
		assert.Len(t, secrets.owners, 2)
		assert.Equal(t, 2, secrets.countForOwner(src.ID))
	})

	t.Run("a reference at a field the spec does not mark stays a reference", func(t *testing.T) {
		store, secrets, src := setup(t, specWithoutSecrets)

		dup, err := duplicate(store, secrets, src)
		require.NoError(t, err)

		cfg := configMap(t, dup.Config)
		for field, plain := range map[string]string{"password": "hunter2", "token": "t0k3n"} {
			ref, ok := cfg[field].(string)
			require.True(t, ok)
			assert.NotEqual(t, configMap(t, src.Config)[field], ref)
			assert.Equal(t, secretOwner{ownerType: ownerType, ownerID: dup.ID}, secrets.owners[ref])
			assert.Equal(t, plain, secrets.values[ref])
		}

		assert.Equal(t, "db", cfg["host"])
	})
}

func TestDuplicateSource(t *testing.T) {
	setup := func(t *testing.T) (*srcStore, *fakeSecrets, *pipelineservice.Source, string) {
		t.Helper()

		ws := uuid.New()
		store := newSrcStore()
		secrets := newFakeSecrets()
		connector := store.addConnector(ws, specWithPassword)
		srcID := uuid.New()
		ref := secrets.seed(srcID, "hunter2")
		runtime := `{"cpu":"1"}`
		src := &pipelineservice.Source{
			ID:                 srcID,
			WorkspaceID:        ws,
			Name:               "pg",
			ManagedConnectorID: connector.ID,
			ExecutorGroup:      pipelineservice.DefaultExecutorGroup,
			Config:             `{"host":"db","password":"` + ref + `"}`,
			RuntimeConfig:      &runtime,
		}
		store.sources.records[src.ID] = src

		return store, secrets, src, ref
	}

	t.Run("copies config with its own secrets into the target group", func(t *testing.T) {
		store, secrets, src, ref := setup(t)
		audit := &captureAudit{}

		dup, err := pipelinesources.NewDuplicateSource(store, secrets, audit).Execute(context.Background(), pipelinesources.DuplicateSourceParams{
			ID:            src.ID,
			WorkspaceID:   src.WorkspaceID,
			ExecutorGroup: " us ",
		})
		require.NoError(t, err)

		assert.NotEqual(t, src.ID, dup.ID)
		assert.Equal(t, "us", dup.ExecutorGroup)
		assert.Equal(t, src.Name, dup.Name)
		assert.Equal(t, src.ManagedConnectorID, dup.ManagedConnectorID)
		require.NotNil(t, dup.RuntimeConfig)
		assert.Equal(t, *src.RuntimeConfig, *dup.RuntimeConfig)

		dupRef, ok := configMap(t, dup.Config)["password"].(string)
		require.True(t, ok)
		assert.NotEqual(t, ref, dupRef)
		assert.Equal(t, secretOwner{ownerType: ownerType, ownerID: dup.ID}, secrets.owners[dupRef])

		plain, err := pipelinesecrets.DecryptConfigSecrets(context.Background(), secrets, ownerType, dup.ID, dup.Config)
		require.NoError(t, err)
		assert.JSONEq(t, `{"host":"db","password":"hunter2"}`, plain)

		assert.Equal(t, 1, secrets.countForOwner(src.ID))
		require.Len(t, audit.events, 1)
		assert.Equal(t, "create", audit.events[0].Action)
		assert.Equal(t, src.ID.String(), payloadMap(t, audit.events[0].After)["duplicated_from"])
	})

	t.Run("name overrides the original's", func(t *testing.T) {
		store, secrets, src, _ := setup(t)

		dup, err := pipelinesources.NewDuplicateSource(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinesources.DuplicateSourceParams{
			ID:            src.ID,
			WorkspaceID:   src.WorkspaceID,
			ExecutorGroup: "us",
			Name:          "pg us",
		})
		require.NoError(t, err)
		assert.Equal(t, "pg us", dup.Name)
	})

	t.Run("connections are not copied and the original is untouched", func(t *testing.T) {
		store, secrets, src, _ := setup(t)
		store.connections.records = []*pipelineservice.Connection{{ID: uuid.New(), WorkspaceID: src.WorkspaceID, Name: "c1", SourceID: src.ID}}
		original := *store.sources.records[src.ID]

		dup, err := pipelinesources.NewDuplicateSource(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinesources.DuplicateSourceParams{
			ID:            src.ID,
			WorkspaceID:   src.WorkspaceID,
			ExecutorGroup: "us",
		})
		require.NoError(t, err)

		conns, err := store.Connections().Find(context.Background(), &pipelineservice.ConnectionFilter{SourceID: filter.Equals(dup.ID)})
		require.NoError(t, err)
		assert.Empty(t, conns)
		assert.Equal(t, original, *store.sources.records[src.ID])
	})

	t.Run("another workspace's source is not found", func(t *testing.T) {
		store, secrets, src, _ := setup(t)

		_, err := pipelinesources.NewDuplicateSource(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinesources.DuplicateSourceParams{
			ID:            src.ID,
			WorkspaceID:   uuid.New(),
			ExecutorGroup: "us",
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
		assert.Len(t, store.sources.records, 1)
	})

	t.Run("invalid group is a validation error", func(t *testing.T) {
		store, secrets, src, _ := setup(t)

		_, err := pipelinesources.NewDuplicateSource(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinesources.DuplicateSourceParams{
			ID:            src.ID,
			WorkspaceID:   src.WorkspaceID,
			ExecutorGroup: "a\nb",
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
	})
}
