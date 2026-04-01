package secretservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RetrieveSecret decrypts and returns a secret value from the database.
type RetrieveSecret struct {
	storage     Storage
	masterKey   []byte
	previousKey []byte
}

// NewRetrieveSecret creates a new RetrieveSecret use case.
// previousKey can be nil if key rotation is not in use.
func NewRetrieveSecret(storage Storage, masterKey, previousKey []byte) *RetrieveSecret {
	return &RetrieveSecret{
		storage:     storage,
		masterKey:   masterKey,
		previousKey: previousKey,
	}
}

// RetrieveSecretParams contains the parameters for retrieving a secret.
type RetrieveSecretParams struct {
	SecretRef string
}

// Execute retrieves and decrypts a secret by its reference.
// Supports lazy key rotation: if decryption with the current key fails and a
// previous key is configured, tries the previous key. On success with the
// previous key, re-encrypts with the current key and updates the DB record.
func (uc *RetrieveSecret) Execute(ctx context.Context, params RetrieveSecretParams) (string, error) {
	id, err := ExtractSecretID(params.SecretRef)
	if err != nil {
		return "", fmt.Errorf("invalid secret reference %q: %w", params.SecretRef, err)
	}

	secret, err := uc.storage.Secrets().First(ctx, &SecretFilter{
		ID: filter.Equals(id),
	})
	if err != nil {
		return "", fmt.Errorf("secret not found: %s: %w", params.SecretRef, err)
	}

	// Try current key
	plaintext, err := Decrypt(uc.masterKey, secret.Salt, secret.Nonce, secret.EncryptedValue)
	if err == nil {
		return string(plaintext), nil
	}

	// Try previous key for lazy rotation
	if uc.previousKey != nil {
		plaintext, prevErr := Decrypt(uc.previousKey, secret.Salt, secret.Nonce, secret.EncryptedValue)
		if prevErr != nil {
			return "", errors.New("decryption failed: invalid key or corrupted data")
		}

		// Re-encrypt with current key
		newCiphertext, newSalt, newNonce, encErr := Encrypt(uc.masterKey, plaintext)
		if encErr != nil {
			return "", fmt.Errorf("re-encrypting during key rotation: %w", encErr)
		}

		// Update DB record
		secret.EncryptedValue = newCiphertext
		secret.Salt = newSalt
		secret.Nonce = newNonce
		secret.KeyVersion++

		if _, err := uc.storage.Secrets().Update(ctx, secret); err != nil {
			return "", fmt.Errorf("updating rotated secret: %w", err)
		}

		return string(plaintext), nil
	}

	return "", errors.New("decryption failed: invalid key or corrupted data")
}
