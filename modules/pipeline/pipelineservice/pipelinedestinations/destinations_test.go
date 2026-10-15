package pipelinedestinations_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
)

func TestGetDestination_WorkspaceScoping(t *testing.T) {
	ws := uuid.New()

	t.Run("returns destination inside the workspace", func(t *testing.T) {
		dest := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, Name: "bq"}
		store := newDestStore(dest)

		uc := pipelinedestinations.NewGetDestination(store)
		got, err := uc.Execute(context.Background(), pipelinedestinations.GetDestinationParams{
			ID:          dest.ID,
			WorkspaceID: ws,
		})
		require.NoError(t, err)
		assert.Equal(t, "bq", got.Name)
	})

	t.Run("refuses cross-workspace lookup", func(t *testing.T) {
		dest := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: uuid.New()}
		store := newDestStore(dest)

		uc := pipelinedestinations.NewGetDestination(store)
		_, err := uc.Execute(context.Background(), pipelinedestinations.GetDestinationParams{
			ID:          dest.ID,
			WorkspaceID: ws,
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
	})

	t.Run("returns not found for an unknown id", func(t *testing.T) {
		store := newDestStore()

		uc := pipelinedestinations.NewGetDestination(store)
		_, err := uc.Execute(context.Background(), pipelinedestinations.GetDestinationParams{
			ID:          uuid.New(),
			WorkspaceID: ws,
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
	})
}

func TestListDestinations_WorkspaceScoping(t *testing.T) {
	t.Run("returns only destinations within the workspace", func(t *testing.T) {
		ws1, ws2 := uuid.New(), uuid.New()
		store := newDestStore(
			&pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws1, Name: "a"},
			&pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws1, Name: "b"},
			&pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws2, Name: "other"},
		)

		uc := pipelinedestinations.NewListDestinations(store)
		got, err := uc.Execute(context.Background(), pipelinedestinations.ListDestinationsParams{WorkspaceID: ws1})
		require.NoError(t, err)
		assert.Len(t, got, 2)

		for _, s := range got {
			assert.Equal(t, ws1, s.WorkspaceID)
		}
	})

	t.Run("returns empty slice for an empty workspace", func(t *testing.T) {
		store := newDestStore()

		uc := pipelinedestinations.NewListDestinations(store)
		got, err := uc.Execute(context.Background(), pipelinedestinations.ListDestinationsParams{WorkspaceID: uuid.New()})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
