package secretservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// DeleteSecret removes a secret from the database.
type DeleteSecret struct {
	storage Storage
}

// NewDeleteSecret creates a new DeleteSecret use case.
func NewDeleteSecret(storage Storage) *DeleteSecret {
	return &DeleteSecret{storage: storage}
}

// Execute deletes a single secret by its reference.
func (uc *DeleteSecret) Execute(ctx context.Context, secretRef string) error {
	id, err := ExtractSecretID(secretRef)
	if err != nil {
		return fmt.Errorf("invalid secret reference %q: %w", secretRef, err)
	}

	if err := uc.storage.Secrets().Delete(ctx, &SecretFilter{
		ID: filter.Equals(id),
	}); err != nil {
		return fmt.Errorf("deleting secret: %w", err)
	}

	return nil
}

// DeleteByOwner deletes all secrets belonging to an owner.
func (uc *DeleteSecret) DeleteByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error {
	if err := uc.storage.Secrets().Delete(ctx, &SecretFilter{
		OwnerType: filter.Equals(ownerType),
		OwnerID:   filter.Equals(ownerID),
	}); err != nil {
		return fmt.Errorf("deleting secrets by owner: %w", err)
	}

	return nil
}
