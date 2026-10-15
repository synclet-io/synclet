package pipelinesources_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

func TestGetSource_WorkspaceScoping(t *testing.T) {
	ws := uuid.New()

	t.Run("returns source inside the workspace", func(t *testing.T) {
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, Name: "pg"}
		store := newSrcStore(src)

		uc := pipelinesources.NewGetSource(store)
		got, err := uc.Execute(context.Background(), pipelinesources.GetSourceParams{
			ID:          src.ID,
			WorkspaceID: ws,
		})
		require.NoError(t, err)
		assert.Equal(t, "pg", got.Name)
	})

	t.Run("refuses cross-workspace lookup", func(t *testing.T) {
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: uuid.New()}
		store := newSrcStore(src)

		uc := pipelinesources.NewGetSource(store)
		_, err := uc.Execute(context.Background(), pipelinesources.GetSourceParams{
			ID:          src.ID,
			WorkspaceID: ws,
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})

	t.Run("returns not found for an unknown id", func(t *testing.T) {
		store := newSrcStore()

		uc := pipelinesources.NewGetSource(store)
		_, err := uc.Execute(context.Background(), pipelinesources.GetSourceParams{
			ID:          uuid.New(),
			WorkspaceID: ws,
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})
}

func TestListSources_WorkspaceScoping(t *testing.T) {
	t.Run("returns only sources within the workspace", func(t *testing.T) {
		ws1, ws2 := uuid.New(), uuid.New()
		store := newSrcStore(
			&pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws1, Name: "a"},
			&pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws1, Name: "b"},
			&pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws2, Name: "other"},
		)

		uc := pipelinesources.NewListSources(store)
		got, err := uc.Execute(context.Background(), pipelinesources.ListSourcesParams{WorkspaceID: ws1})
		require.NoError(t, err)
		assert.Len(t, got, 2)

		for _, s := range got {
			assert.Equal(t, ws1, s.WorkspaceID)
		}
	})

	t.Run("returns empty slice for an empty workspace", func(t *testing.T) {
		store := newSrcStore()

		uc := pipelinesources.NewListSources(store)
		got, err := uc.Execute(context.Background(), pipelinesources.ListSourcesParams{WorkspaceID: uuid.New()})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
