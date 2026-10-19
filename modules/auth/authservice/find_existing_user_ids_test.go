package authservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserIDStorage struct {
	existing map[uuid.UUID]bool
	calls    [][]uuid.UUID
}

func (f *fakeUserIDStorage) ExistingUserIDs(_ context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.calls = append(f.calls, ids)

	found := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if f.existing[id] {
			found = append(found, id)
		}
	}

	return found, nil
}

func TestFindExistingUserIDs_EmptyInputSkipsStorage(t *testing.T) {
	users := &fakeUserIDStorage{}

	got, err := NewFindExistingUserIDs(users).Execute(context.Background(), nil)
	require.NoError(t, err)

	assert.Empty(t, got)
	assert.Empty(t, users.calls)
}

func TestFindExistingUserIDs_ReturnsExistingSubset(t *testing.T) {
	alice, bob, gone := uuid.New(), uuid.New(), uuid.New()
	users := &fakeUserIDStorage{existing: map[uuid.UUID]bool{alice: true, bob: true}}

	got, err := NewFindExistingUserIDs(users).Execute(context.Background(), []uuid.UUID{alice, gone, bob})
	require.NoError(t, err)

	assert.ElementsMatch(t, []uuid.UUID{alice, bob}, got)
	require.Len(t, users.calls, 1)
	assert.ElementsMatch(t, []uuid.UUID{alice, gone, bob}, users.calls[0])
}
