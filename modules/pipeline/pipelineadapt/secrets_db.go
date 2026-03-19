package pipelineadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/secret/secretservice"
)

// DBSecretsProvider implements pipelineservice.SecretsProvider using the
// DB-backed encryption use cases.
type DBSecretsProvider struct {
	storeSecret    *secretservice.StoreSecret
	retrieveSecret *secretservice.RetrieveSecret
	deleteSecret   *secretservice.DeleteSecret
}

// NewDBSecretsProvider creates a new DBSecretsProvider.
func NewDBSecretsProvider(
	store *secretservice.StoreSecret,
	retrieve *secretservice.RetrieveSecret,
	del *secretservice.DeleteSecret,
) *DBSecretsProvider {
	return &DBSecretsProvider{
		storeSecret:    store,
		retrieveSecret: retrieve,
		deleteSecret:   del,
	}
}

// StoreSecret encrypts and stores a secret value.
func (p *DBSecretsProvider) StoreSecret(ctx context.Context, ownerType string, ownerID uuid.UUID, plaintext string) (string, error) {
	return p.storeSecret.Execute(ctx, secretservice.StoreSecretParams{
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Plaintext: plaintext,
	})
}

// RetrieveSecret decrypts and returns a secret value.
func (p *DBSecretsProvider) RetrieveSecret(ctx context.Context, secretRef string) (string, error) {
	return p.retrieveSecret.Execute(ctx, secretservice.RetrieveSecretParams{
		SecretRef: secretRef,
	})
}

// DeleteSecret removes a single secret.
func (p *DBSecretsProvider) DeleteSecret(ctx context.Context, secretRef string) error {
	return p.deleteSecret.Execute(ctx, secretRef)
}

// DeleteByOwner removes all secrets for an owner.
func (p *DBSecretsProvider) DeleteByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error {
	return p.deleteSecret.DeleteByOwner(ctx, ownerType, ownerID)
}
