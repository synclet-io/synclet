package pipelinedestinations_test

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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
)

type destStore struct {
	pipelineservice.Storage
	dests *destStorage
}

func (s *destStore) Destinations() pipelineservice.DestinationsStorage {
	return s.dests
}

type destStorage struct {
	pipelineservice.DestinationsStorage
	records map[uuid.UUID]*pipelineservice.Destination
}

func (s *destStorage) First(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Destination, error) {
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

	return nil, pipelineservice.ErrDestinationNotFound
}

func (s *destStorage) Find(_ context.Context, f *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	out := make([]*pipelineservice.Destination, 0, len(s.records))

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

func newDestStore(records ...*pipelineservice.Destination) *destStore {
	m := make(map[uuid.UUID]*pipelineservice.Destination, len(records))
	for _, r := range records {
		m[r.ID] = r
	}

	return &destStore{dests: &destStorage{records: m}}
}

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
		require.Error(t, err)
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

		for _, d := range got {
			assert.Equal(t, ws1, d.WorkspaceID)
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
