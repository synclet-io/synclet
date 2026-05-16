package secretservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreSecret_PersistsCiphertextAndReturnsRef(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	uc := NewStoreSecret(store, Config{MasterKey: randomKey(t), KeyVersion: 1})

	ownerID := uuid.New()
	plaintext := "very-secret-token"

	ref, err := uc.Execute(ctx, StoreSecretParams{
		OwnerType: "connector",
		OwnerID:   ownerID,
		Plaintext: plaintext,
	})
	require.NoError(t, err)
	require.NotEmpty(t, ref)
	assert.True(t, IsSecretRef(ref), "returned reference must be a secret ref")

	rows, err := store.Secrets().Find(ctx, &SecretFilter{})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	row := rows[0]
	assert.Equal(t, ownerID, row.OwnerID)
	assert.Equal(t, "connector", row.OwnerType)
	assert.Equal(t, 1, row.KeyVersion)
	assert.NotEqual(t, []byte(plaintext), row.EncryptedValue, "stored value must be encrypted, not plaintext")
	assert.NotEmpty(t, row.Salt)
	assert.NotEmpty(t, row.Nonce)
}

func TestStoreSecret_TwoCallsProduceDistinctSecrets(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	uc := NewStoreSecret(store, Config{MasterKey: randomKey(t), KeyVersion: 1})

	refA, err := uc.Execute(ctx, StoreSecretParams{OwnerType: "x", OwnerID: uuid.New(), Plaintext: "same"})
	require.NoError(t, err)
	refB, err := uc.Execute(ctx, StoreSecretParams{OwnerType: "x", OwnerID: uuid.New(), Plaintext: "same"})
	require.NoError(t, err)

	assert.NotEqual(t, refA, refB, "each call must produce a distinct secret reference")

	rows, err := store.Secrets().Find(ctx, &SecretFilter{})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.NotEqual(t, rows[0].EncryptedValue, rows[1].EncryptedValue, "ciphertexts must differ across calls")
}
