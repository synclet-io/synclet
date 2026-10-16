package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type purgeKeysStorage struct {
	Storage
	keys *purgeKeys
}

func (s *purgeKeysStorage) APIKeys() APIKeysStorage { return s.keys }

type purgeKeys struct {
	APIKeysStorage
	filters []*APIKeyFilter
	err     error
}

func (k *purgeKeys) Delete(_ context.Context, f *APIKeyFilter) error {
	k.filters = append(k.filters, f)

	return k.err
}

func TestPurgeWorkspaceAPIKeys(t *testing.T) {
	workspaceID := uuid.New()

	t.Run("deletes keys scoped to the workspace only", func(t *testing.T) {
		keys := &purgeKeys{}

		require.NoError(t, NewPurgeWorkspaceAPIKeys(&purgeKeysStorage{keys: keys}).Execute(context.Background(), workspaceID))

		require.Len(t, keys.filters, 1)
		assert.Equal(t, &APIKeyFilter{WorkspaceID: filter.Equals(workspaceID)}, keys.filters[0])
	})

	t.Run("propagates storage errors", func(t *testing.T) {
		errBoom := errors.New("boom")
		keys := &purgeKeys{err: errBoom}

		require.ErrorIs(t, NewPurgeWorkspaceAPIKeys(&purgeKeysStorage{keys: keys}).Execute(context.Background(), workspaceID), errBoom)
	})
}
