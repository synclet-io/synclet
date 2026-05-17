package pipelinesources_test

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

type srcStore struct {
	pipelineservice.Storage
	sources *srcStorage
}

func (s *srcStore) Sources() pipelineservice.SourcesStorage {
	return s.sources
}

type srcStorage struct {
	pipelineservice.SourcesStorage
	records map[uuid.UUID]*pipelineservice.Source
}

func (s *srcStorage) First(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	for _, rec := range s.records {
		if f.ID != nil {
			if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.ID != eq.Value {
				continue
			}
		}

		if f.WorkspaceID != nil {
			if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.WorkspaceID != eq.Value {
				continue
			}
		}

		cp := *rec

		return &cp, nil
	}

	return nil, pipelineservice.ErrSourceNotFound
}

func (s *srcStorage) Find(_ context.Context, f *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	out := make([]*pipelineservice.Source, 0, len(s.records))

	for _, rec := range s.records {
		if f.WorkspaceID != nil {
			if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.WorkspaceID != eq.Value {
				continue
			}
		}

		cp := *rec
		out = append(out, &cp)
	}

	return out, nil
}

func newSrcStore(records ...*pipelineservice.Source) *srcStore {
	m := make(map[uuid.UUID]*pipelineservice.Source, len(records))
	for _, r := range records {
		m[r.ID] = r
	}

	return &srcStore{sources: &srcStorage{records: m}}
}

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
		require.Error(t, err)
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
