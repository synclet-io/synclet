package pipelinedestinations_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
)

func TestCreateDestinationExecutorGroup(t *testing.T) {
	cases := []struct {
		name  string
		group string
		want  string
	}{
		{name: "trimmed and kept verbatim", group: "  Prod Cluster ACME  ", want: "Prod Cluster ACME"},
		{name: "empty is default", group: "", want: pipelineservice.DefaultExecutorGroup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := uuid.New()
			store := newDestStore()
			conn := store.addConnector(ws, specWithoutSecrets)

			created, err := pipelinedestinations.NewCreateDestination(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
				WorkspaceID:        ws,
				Name:               "x",
				ManagedConnectorID: conn.ID,
				Config:             json.RawMessage(`{}`),
				ExecutorGroup:      tc.group,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, created.ExecutorGroup)
		})
	}

	t.Run("invalid group is a validation error", func(t *testing.T) {
		ws := uuid.New()
		store := newDestStore()
		conn := store.addConnector(ws, specWithoutSecrets)

		_, err := pipelinedestinations.NewCreateDestination(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        ws,
			Name:               "x",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{}`),
			ExecutorGroup:      "a\nb",
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
	})
}

func TestUpdateDestinationExecutorGroup(t *testing.T) {
	setup := func(t *testing.T) (*destStore, *pipelineservice.Destination) {
		t.Helper()

		ws := uuid.New()
		store := newDestStore()
		connector := store.addConnector(ws, specWithoutSecrets)
		dest := &pipelineservice.Destination{
			ID:                 uuid.New(),
			WorkspaceID:        ws,
			Name:               "pg",
			ManagedConnectorID: connector.ID,
			ExecutorGroup:      pipelineservice.DefaultExecutorGroup,
			Config:             `{}`,
		}
		store.destinations.records[dest.ID] = dest

		return store, dest
	}

	update := func(store *destStore, dest *pipelineservice.Destination, group string) (*pipelineservice.Destination, error) {
		return pipelinedestinations.NewUpdateDestination(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinedestinations.UpdateDestinationParams{
			ID:            dest.ID,
			WorkspaceID:   dest.WorkspaceID,
			ExecutorGroup: &group,
		})
	}

	t.Run("no connections moves the destination", func(t *testing.T) {
		store, dest := setup(t)

		updated, err := update(store, dest, " us ")
		require.NoError(t, err)
		assert.Equal(t, "us", updated.ExecutorGroup)
		assert.Equal(t, "us", store.destinations.records[dest.ID].ExecutorGroup)
	})

	t.Run("connected destination refuses a group change", func(t *testing.T) {
		store, dest := setup(t)
		store.connections.records = []*pipelineservice.Connection{{ID: uuid.New(), WorkspaceID: dest.WorkspaceID, Name: "c1", DestinationID: dest.ID}}

		_, err := update(store, dest, "us")

		var inUse *pipelineservice.InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, []string{"c1"}, inUse.ConnectionNames)
		assert.Equal(t, pipelineservice.DefaultExecutorGroup, store.destinations.records[dest.ID].ExecutorGroup)
	})

	t.Run("connected destination accepts its own group", func(t *testing.T) {
		store, dest := setup(t)
		store.connections.records = []*pipelineservice.Connection{{ID: uuid.New(), WorkspaceID: dest.WorkspaceID, Name: "c1", DestinationID: dest.ID}}

		updated, err := update(store, dest, "")
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.DefaultExecutorGroup, updated.ExecutorGroup)
	})

	t.Run("invalid group is a validation error", func(t *testing.T) {
		store, dest := setup(t)

		_, err := update(store, dest, "a\nb")

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
	})
}
