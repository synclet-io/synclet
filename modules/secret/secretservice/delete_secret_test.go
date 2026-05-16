package secretservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteSecret_RemovesByRef(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	store := newFakeStorage()
	store.secrets = append(store.secrets, &Secret{ID: id})

	require.NoError(t, NewDeleteSecret(store).Execute(ctx, MakeSecretRef(id)))

	rows, err := store.Secrets().Find(ctx, &SecretFilter{})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestDeleteSecret_InvalidRefRejected(t *testing.T) {
	ctx := context.Background()
	err := NewDeleteSecret(newFakeStorage()).Execute(ctx, "not-a-ref")
	require.Error(t, err)
}

func TestDeleteSecret_IdempotentWhenMissing(t *testing.T) {
	ctx := context.Background()
	err := NewDeleteSecret(newFakeStorage()).Execute(ctx, MakeSecretRef(uuid.New()))
	assert.NoError(t, err)
}

func TestDeleteSecret_DeleteByOwnerRemovesAllMatching(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	ownerID := uuid.New()
	store.secrets = append(store.secrets,
		&Secret{ID: uuid.New(), OwnerType: "channel", OwnerID: ownerID},
		&Secret{ID: uuid.New(), OwnerType: "channel", OwnerID: ownerID},
		&Secret{ID: uuid.New(), OwnerType: "channel", OwnerID: uuid.New()},
	)

	require.NoError(t, NewDeleteSecret(store).DeleteByOwner(ctx, "channel", ownerID))

	rows, err := store.Secrets().Find(ctx, &SecretFilter{})
	require.NoError(t, err)
	assert.Len(t, rows, 1, "only secrets for the targeted owner are removed")
}
