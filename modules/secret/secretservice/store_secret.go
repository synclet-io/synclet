package secretservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// StoreSecret encrypts and stores a secret value in the database.
type StoreSecret struct {
	storage    Storage
	masterKey  []byte
	keyVersion int
}

// NewStoreSecret creates a new StoreSecret use case.
func NewStoreSecret(storage Storage, masterKey []byte, keyVersion int) *StoreSecret {
	return &StoreSecret{
		storage:    storage,
		masterKey:  masterKey,
		keyVersion: keyVersion,
	}
}

// StoreSecretParams contains the parameters for storing a secret.
type StoreSecretParams struct {
	OwnerType string
	OwnerID   uuid.UUID
	Plaintext string
}

// Execute encrypts and stores the secret, returning a secret reference.
func (uc *StoreSecret) Execute(ctx context.Context, params StoreSecretParams) (string, error) {
	secretID := uuid.New()

	ciphertext, salt, nonce, err := Encrypt(uc.masterKey, []byte(params.Plaintext))
	if err != nil {
		return "", fmt.Errorf("encrypting secret: %w", err)
	}

	secret := &Secret{
		ID:             secretID,
		EncryptedValue: ciphertext,
		Salt:           salt,
		Nonce:          nonce,
		KeyVersion:     uc.keyVersion,
		OwnerType:      params.OwnerType,
		OwnerID:        params.OwnerID,
	}

	if _, err := uc.storage.Secrets().Create(ctx, secret); err != nil {
		return "", fmt.Errorf("storing secret: %w", err)
	}

	return MakeSecretRef(secret.ID), nil
}
