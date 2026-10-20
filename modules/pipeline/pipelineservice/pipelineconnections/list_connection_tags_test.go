package pipelineconnections_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

func (s *createConnConnectionsStorage) ListTagCounts(_ context.Context, workspaceID uuid.UUID) ([]pipelineservice.TagCount, error) {
	counts := map[string]int{}

	for _, conn := range s.records {
		if conn.WorkspaceID != workspaceID {
			continue
		}

		for _, tag := range conn.Tags {
			counts[tag]++
		}
	}

	out := make([]pipelineservice.TagCount, 0, len(counts))
	for tag, count := range counts {
		out = append(out, pipelineservice.TagCount{Tag: tag, ConnectionCount: count})
	}

	slices.SortFunc(out, func(a, b pipelineservice.TagCount) int {
		if a.Tag < b.Tag {
			return -1
		}

		if a.Tag > b.Tag {
			return 1
		}

		return 0
	})

	return out, nil
}

func TestListConnectionTags(t *testing.T) {
	t.Run("counts the caller's workspace tags only", func(t *testing.T) {
		ws := uuid.New()
		otherWs := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Tags: []string{"finance", "prod"}})
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Tags: []string{"prod"}})
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws})
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: otherWs, Tags: []string{"prod", "secret"}})

		uc := pipelineconnections.NewListConnectionTags(store)
		got, err := uc.Execute(context.Background(), pipelineconnections.ListConnectionTagsParams{WorkspaceID: ws})
		require.NoError(t, err)
		assert.Equal(t, []pipelineservice.TagCount{
			{Tag: "finance", ConnectionCount: 1},
			{Tag: "prod", ConnectionCount: 2},
		}, got)
	})

	t.Run("returns an empty list for a workspace without tags", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New(), Tags: []string{"prod"}})

		uc := pipelineconnections.NewListConnectionTags(store)
		got, err := uc.Execute(context.Background(), pipelineconnections.ListConnectionTagsParams{WorkspaceID: ws})
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
