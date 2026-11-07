package pipelinesources_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

func TestCreateSourceExecutorGroup(t *testing.T) {
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
			store := newSrcStore()
			conn := store.addConnector(ws, specWithoutSecrets)

			created, err := pipelinesources.NewCreateSource(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.CreateSourceParams{
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
		store := newSrcStore()
		conn := store.addConnector(ws, specWithoutSecrets)

		_, err := pipelinesources.NewCreateSource(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.CreateSourceParams{
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

func TestUpdateSourceExecutorGroup(t *testing.T) {
	setup := func(t *testing.T) (*srcStore, *pipelineservice.Source) {
		t.Helper()

		ws := uuid.New()
		store := newSrcStore()
		connector := store.addConnector(ws, specWithoutSecrets)
		src := &pipelineservice.Source{
			ID:                 uuid.New(),
			WorkspaceID:        ws,
			Name:               "pg",
			ManagedConnectorID: connector.ID,
			ExecutorGroup:      pipelineservice.DefaultExecutorGroup,
			Config:             `{}`,
		}
		store.sources.records[src.ID] = src

		return store, src
	}

	update := func(store *srcStore, src *pipelineservice.Source, group string) (*pipelineservice.Source, error) {
		return pipelinesources.NewUpdateSource(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.UpdateSourceParams{
			ID:            src.ID,
			WorkspaceID:   src.WorkspaceID,
			ExecutorGroup: &group,
		})
	}

	t.Run("no connections moves the source", func(t *testing.T) {
		store, src := setup(t)

		updated, err := update(store, src, " us ")
		require.NoError(t, err)
		assert.Equal(t, "us", updated.ExecutorGroup)
		assert.Equal(t, "us", store.sources.records[src.ID].ExecutorGroup)
	})

	t.Run("connected source refuses a group change", func(t *testing.T) {
		store, src := setup(t)
		store.connections.records = []*pipelineservice.Connection{{ID: uuid.New(), WorkspaceID: src.WorkspaceID, Name: "c1", SourceID: src.ID}}

		_, err := update(store, src, "us")

		var inUse *pipelineservice.InUseError
		require.ErrorAs(t, err, &inUse)
		assert.Equal(t, []string{"c1"}, inUse.ConnectionNames)
		assert.Equal(t, pipelineservice.DefaultExecutorGroup, store.sources.records[src.ID].ExecutorGroup)
	})

	t.Run("connected source accepts its own group", func(t *testing.T) {
		store, src := setup(t)
		store.connections.records = []*pipelineservice.Connection{{ID: uuid.New(), WorkspaceID: src.WorkspaceID, Name: "c1", SourceID: src.ID}}

		updated, err := update(store, src, "")
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.DefaultExecutorGroup, updated.ExecutorGroup)
	})

	t.Run("invalid group is a validation error", func(t *testing.T) {
		store, src := setup(t)

		_, err := update(store, src, "a\nb")

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
	})
}
