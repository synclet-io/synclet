package pipelinedestinations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

const specWithTwoSecrets = `{"connectionSpecification":{"type":"object","properties":{
	"host":{"type":"string"},
	"password":{"type":"string","airbyte_secret":true},
	"token":{"type":"string","airbyte_secret":true}}}}`

func TestDuplicateDestinationSecrets(t *testing.T) {
	setup := func(t *testing.T, spec string) (*destStore, *fakeSecrets, *pipelineservice.Destination) {
		t.Helper()

		ws := uuid.New()
		store := newDestStore()
		secrets := newFakeSecrets()
		connector := store.addConnector(ws, spec)
		destID := uuid.New()
		dest := &pipelineservice.Destination{
			ID:                 destID,
			WorkspaceID:        ws,
			Name:               "pg",
			ManagedConnectorID: connector.ID,
			Config:             `{"host":"db","password":"` + secrets.seed(destID, "hunter2") + `","token":"` + secrets.seed(destID, "t0k3n") + `"}`,
		}
		store.destinations.records[dest.ID] = dest

		return store, secrets, dest
	}

	duplicate := func(store *destStore, secrets *fakeSecrets, dest *pipelineservice.Destination) (*pipelineservice.Destination, error) {
		return pipelinedestinations.NewDuplicateDestination(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinedestinations.DuplicateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   dest.WorkspaceID,
			ExecutorGroup: "us",
		})
	}

	t.Run("a failed secret store leaves no secrets of the copy", func(t *testing.T) {
		store, secrets, dest := setup(t, specWithTwoSecrets)
		secrets.storeErr = errors.New("vault down")
		secrets.failStoreAfter = 1

		_, err := duplicate(store, secrets, dest)
		require.Error(t, err)
		assert.Len(t, secrets.owners, 2)
		assert.Equal(t, 2, secrets.countForOwner(dest.ID))
		assert.Len(t, store.destinations.records, 1)
	})

	t.Run("a failed create leaves no secrets of the copy", func(t *testing.T) {
		store, secrets, dest := setup(t, specWithTwoSecrets)
		store.destinations.createErr = errors.New("db down")

		_, err := duplicate(store, secrets, dest)
		require.Error(t, err)
		assert.Len(t, secrets.owners, 2)
		assert.Equal(t, 2, secrets.countForOwner(dest.ID))
	})

	t.Run("a reference at a field the spec does not mark stays a reference", func(t *testing.T) {
		store, secrets, dest := setup(t, specWithoutSecrets)

		dup, err := duplicate(store, secrets, dest)
		require.NoError(t, err)

		cfg := configMap(t, dup.Config)
		for field, plain := range map[string]string{"password": "hunter2", "token": "t0k3n"} {
			ref, ok := cfg[field].(string)
			require.True(t, ok)
			assert.NotEqual(t, configMap(t, dest.Config)[field], ref)
			assert.Equal(t, secretOwner{ownerType: ownerType, ownerID: dup.ID}, secrets.owners[ref])
			assert.Equal(t, plain, secrets.values[ref])
		}

		assert.Equal(t, "db", cfg["host"])
	})
}

func TestDuplicateDestination(t *testing.T) {
	setup := func(t *testing.T) (*destStore, *fakeSecrets, *pipelineservice.Destination, string) {
		t.Helper()

		ws := uuid.New()
		store := newDestStore()
		secrets := newFakeSecrets()
		connector := store.addConnector(ws, specWithPassword)
		destID := uuid.New()
		ref := secrets.seed(destID, "hunter2")
		runtime := `{"cpu":"1"}`
		dest := &pipelineservice.Destination{
			ID:                 destID,
			WorkspaceID:        ws,
			Name:               "pg",
			ManagedConnectorID: connector.ID,
			ExecutorGroup:      pipelineservice.DefaultExecutorGroup,
			Config:             `{"host":"db","password":"` + ref + `"}`,
			RuntimeConfig:      &runtime,
		}
		store.destinations.records[dest.ID] = dest

		return store, secrets, dest, ref
	}

	t.Run("copies config with its own secrets into the target group", func(t *testing.T) {
		store, secrets, dest, ref := setup(t)
		audit := &captureAudit{}

		dup, err := pipelinedestinations.NewDuplicateDestination(store, secrets, audit).Execute(context.Background(), pipelinedestinations.DuplicateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   dest.WorkspaceID,
			ExecutorGroup: " us ",
		})
		require.NoError(t, err)

		assert.NotEqual(t, dest.ID, dup.ID)
		assert.Equal(t, "us", dup.ExecutorGroup)
		assert.Equal(t, dest.Name, dup.Name)
		assert.Equal(t, dest.ManagedConnectorID, dup.ManagedConnectorID)
		require.NotNil(t, dup.RuntimeConfig)
		assert.Equal(t, *dest.RuntimeConfig, *dup.RuntimeConfig)

		dupRef, ok := configMap(t, dup.Config)["password"].(string)
		require.True(t, ok)
		assert.NotEqual(t, ref, dupRef)
		assert.Equal(t, secretOwner{ownerType: ownerType, ownerID: dup.ID}, secrets.owners[dupRef])

		plain, err := pipelinesecrets.DecryptConfigSecrets(context.Background(), secrets, ownerType, dup.ID, dup.Config)
		require.NoError(t, err)
		assert.JSONEq(t, `{"host":"db","password":"hunter2"}`, plain)

		assert.Equal(t, 1, secrets.countForOwner(dest.ID))
		require.Len(t, audit.events, 1)
		assert.Equal(t, "create", audit.events[0].Action)
		assert.Equal(t, dest.ID.String(), payloadMap(t, audit.events[0].After)["duplicated_from"])
	})

	t.Run("name overrides the original's", func(t *testing.T) {
		store, secrets, dest, _ := setup(t)

		dup, err := pipelinedestinations.NewDuplicateDestination(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinedestinations.DuplicateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   dest.WorkspaceID,
			ExecutorGroup: "us",
			Name:          "pg us",
		})
		require.NoError(t, err)
		assert.Equal(t, "pg us", dup.Name)
	})

	t.Run("connections are not copied and the original is untouched", func(t *testing.T) {
		store, secrets, dest, _ := setup(t)
		store.connections.records = []*pipelineservice.Connection{{ID: uuid.New(), WorkspaceID: dest.WorkspaceID, Name: "c1", DestinationID: dest.ID}}
		original := *store.destinations.records[dest.ID]

		dup, err := pipelinedestinations.NewDuplicateDestination(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinedestinations.DuplicateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   dest.WorkspaceID,
			ExecutorGroup: "us",
		})
		require.NoError(t, err)

		conns, err := store.Connections().Find(context.Background(), &pipelineservice.ConnectionFilter{DestinationID: filter.Equals(dup.ID)})
		require.NoError(t, err)
		assert.Empty(t, conns)
		assert.Equal(t, original, *store.destinations.records[dest.ID])
	})

	t.Run("another workspace's destination is not found", func(t *testing.T) {
		store, secrets, dest, _ := setup(t)

		_, err := pipelinedestinations.NewDuplicateDestination(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinedestinations.DuplicateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   uuid.New(),
			ExecutorGroup: "us",
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
		assert.Len(t, store.destinations.records, 1)
	})

	t.Run("invalid group is a validation error", func(t *testing.T) {
		store, secrets, dest, _ := setup(t)

		_, err := pipelinedestinations.NewDuplicateDestination(store, secrets, &captureAudit{}).Execute(context.Background(), pipelinedestinations.DuplicateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   dest.WorkspaceID,
			ExecutorGroup: "a\nb",
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
	})
}
