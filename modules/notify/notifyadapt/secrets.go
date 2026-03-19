package notifyadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/secret/secretservice"
)

// SecretsAdapter implements notifyservice.SecretsProvider using the secret module.
type SecretsAdapter struct {
	store    *secretservice.StoreSecret
	retrieve *secretservice.RetrieveSecret
	delete   *secretservice.DeleteSecret
}

// NewSecretsAdapter creates a new SecretsAdapter.
func NewSecretsAdapter(
	store *secretservice.StoreSecret,
	retrieve *secretservice.RetrieveSecret,
	del *secretservice.DeleteSecret,
) *SecretsAdapter {
	return &SecretsAdapter{store: store, retrieve: retrieve, delete: del}
}

// StoreSecret encrypts and stores a secret value.
func (a *SecretsAdapter) StoreSecret(ctx context.Context, ownerType string, ownerID uuid.UUID, plaintext string) (string, error) {
	return a.store.Execute(ctx, secretservice.StoreSecretParams{
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Plaintext: plaintext,
	})
}

// RetrieveSecret decrypts and returns a secret value.
func (a *SecretsAdapter) RetrieveSecret(ctx context.Context, secretRef string) (string, error) {
	return a.retrieve.Execute(ctx, secretservice.RetrieveSecretParams{SecretRef: secretRef})
}

// DeleteSecret removes a single secret.
func (a *SecretsAdapter) DeleteSecret(ctx context.Context, secretRef string) error {
	return a.delete.Execute(ctx, secretRef)
}

// DeleteByOwner removes all secrets for an owner.
func (a *SecretsAdapter) DeleteByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error {
	return a.delete.DeleteByOwner(ctx, ownerType, ownerID)
}
