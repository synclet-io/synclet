package secretservice

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSecretsStorage is an in-memory SecretsStorage used to drive use-case
// tests. Only First and Update are implemented; the rest panic to flag any
// accidental dependency from the use case under test.
type stubSecretsStorage struct {
	secret *Secret
	stored *Secret
}

func (s *stubSecretsStorage) First(_ context.Context, _ *SecretFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*Secret, error) {
	if s.secret == nil {
		return nil, errors.New("not found")
	}

	return s.secret, nil
}

func (s *stubSecretsStorage) Update(_ context.Context, secret *Secret) (*Secret, error) {
	s.stored = secret

	return secret, nil
}

func (s *stubSecretsStorage) Create(_ context.Context, _ *Secret) (*Secret, error) {
	panic("Create not implemented in stub")
}

func (s *stubSecretsStorage) BatchCreate(_ context.Context, _ []*Secret) ([]*Secret, error) {
	panic("BatchCreate not implemented in stub")
}

func (s *stubSecretsStorage) Count(_ context.Context, _ *SecretFilter) (int, error) {
	panic("Count not implemented in stub")
}

func (s *stubSecretsStorage) Save(_ context.Context, _ *Secret) (*Secret, error) {
	panic("Save not implemented in stub")
}

func (s *stubSecretsStorage) FirstOrCreate(_ context.Context, _ *SecretFilter, _ *Secret, _ ...optionutil.Option[dbutil.SelectOptions]) (*Secret, error) {
	panic("FirstOrCreate not implemented in stub")
}

func (s *stubSecretsStorage) Find(_ context.Context, _ *SecretFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*Secret, error) {
	panic("Find not implemented in stub")
}

func (s *stubSecretsStorage) Delete(_ context.Context, _ *SecretFilter) error {
	panic("Delete not implemented in stub")
}

func (s *stubSecretsStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	panic("WithAdvisoryLock not implemented in stub")
}

// stubStorage exposes only Secrets(); the other Storage methods panic.
type stubStorage struct {
	secrets *stubSecretsStorage
}

func (s *stubStorage) Secrets() SecretsStorage { return s.secrets }

func (s *stubStorage) IdempotencyKeys() idempotency.Storage {
	panic("IdempotencyKeys not implemented in stub")
}

func (s *stubStorage) ExecuteInTransaction(_ context.Context, _ func(ctx context.Context, tx Storage) error) error {
	panic("ExecuteInTransaction not implemented in stub")
}

func (s *stubStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error {
	panic("WithAdvisoryLock not implemented in stub")
}

func TestRetrieveSecret_DecryptWithCurrentKey(t *testing.T) {
	ctx := context.Background()
	currentKey := randomKey(t)

	plaintext := []byte("password-123")
	ciphertext, salt, nonce, err := Encrypt(currentKey, plaintext)
	require.NoError(t, err)

	id := uuid.New()
	store := &stubSecretsStorage{
		secret: &Secret{
			ID:             id,
			EncryptedValue: ciphertext,
			Salt:           salt,
			Nonce:          nonce,
			KeyVersion:     2,
		},
	}
	retriever := NewRetrieveSecret(&stubStorage{secrets: store}, Config{
		MasterKey:  currentKey,
		KeyVersion: 2,
	})

	got, err := retriever.Execute(ctx, RetrieveSecretParams{SecretRef: MakeSecretRef(id)})
	require.NoError(t, err)
	assert.Equal(t, string(plaintext), got)
	assert.Nil(t, store.stored, "no re-encrypt expected when current key works")
}

func TestRetrieveSecret_LazyKeyRotation(t *testing.T) {
	// Encrypt with the previous key; configure use case with a new current key.
	// Retrieve should: decrypt via previous key, re-encrypt with the current key,
	// persist the new ciphertext, and bump KeyVersion.
	ctx := context.Background()
	previousKey := randomKey(t)
	currentKey := randomKey(t)

	plaintext := []byte("rotate-me")
	ciphertext, salt, nonce, err := Encrypt(previousKey, plaintext)
	require.NoError(t, err)

	id := uuid.New()
	store := &stubSecretsStorage{
		secret: &Secret{
			ID:             id,
			EncryptedValue: ciphertext,
			Salt:           salt,
			Nonce:          nonce,
			KeyVersion:     1,
		},
	}
	retriever := NewRetrieveSecret(&stubStorage{secrets: store}, Config{
		MasterKey:   currentKey,
		PreviousKey: previousKey,
		KeyVersion:  2,
	})

	got, err := retriever.Execute(ctx, RetrieveSecretParams{SecretRef: MakeSecretRef(id)})
	require.NoError(t, err)
	assert.Equal(t, string(plaintext), got)

	require.NotNil(t, store.stored, "re-encrypted secret must be persisted")
	assert.Equal(t, 2, store.stored.KeyVersion, "KeyVersion must be bumped")
	assert.NotEqual(t, ciphertext, store.stored.EncryptedValue, "ciphertext must be re-encrypted")
	assert.NotEqual(t, salt, store.stored.Salt, "salt must be regenerated")
	assert.NotEqual(t, nonce, store.stored.Nonce, "nonce must be regenerated")

	// The re-encrypted blob must decrypt with the current key.
	roundTrip, err := Decrypt(currentKey, store.stored.Salt, store.stored.Nonce, store.stored.EncryptedValue)
	require.NoError(t, err)
	assert.Equal(t, plaintext, roundTrip)
}

func TestRetrieveSecret_NoPreviousKey_ReturnsDecryptionFailed(t *testing.T) {
	ctx := context.Background()
	wrongKey := randomKey(t)
	encryptionKey := randomKey(t)

	ciphertext, salt, nonce, err := Encrypt(encryptionKey, []byte("payload"))
	require.NoError(t, err)

	id := uuid.New()
	store := &stubSecretsStorage{
		secret: &Secret{
			ID:             id,
			EncryptedValue: ciphertext,
			Salt:           salt,
			Nonce:          nonce,
			KeyVersion:     1,
		},
	}
	retriever := NewRetrieveSecret(&stubStorage{secrets: store}, Config{
		MasterKey:  wrongKey,
		KeyVersion: 1,
	})

	_, err = retriever.Execute(ctx, RetrieveSecretParams{SecretRef: MakeSecretRef(id)})
	require.ErrorIs(t, err, ErrDecryptionFailed)
	assert.Nil(t, store.stored)
}

func TestRetrieveSecret_PreviousKeyAlsoFails_ReturnsDecryptionFailed(t *testing.T) {
	ctx := context.Background()
	encryptionKey := randomKey(t)

	ciphertext, salt, nonce, err := Encrypt(encryptionKey, []byte("payload"))
	require.NoError(t, err)

	id := uuid.New()
	store := &stubSecretsStorage{
		secret: &Secret{
			ID:             id,
			EncryptedValue: ciphertext,
			Salt:           salt,
			Nonce:          nonce,
			KeyVersion:     1,
		},
	}
	retriever := NewRetrieveSecret(&stubStorage{secrets: store}, Config{
		MasterKey:   randomKey(t),
		PreviousKey: randomKey(t),
		KeyVersion:  2,
	})

	_, err = retriever.Execute(ctx, RetrieveSecretParams{SecretRef: MakeSecretRef(id)})
	require.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestRetrieveSecret_InvalidRefRejected(t *testing.T) {
	ctx := context.Background()
	store := &stubSecretsStorage{}
	retriever := NewRetrieveSecret(&stubStorage{secrets: store}, Config{MasterKey: randomKey(t)})

	_, err := retriever.Execute(ctx, RetrieveSecretParams{SecretRef: "not-a-secret-ref"}) //nolint:gosec // test fixture, not a real credential
	require.Error(t, err)
}
