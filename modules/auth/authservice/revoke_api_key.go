package authservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RevokeAPIKey deletes an API key.
type RevokeAPIKey struct {
	storage Storage
	audit   AuditRecorder
}

// NewRevokeAPIKey creates a new RevokeAPIKey use case.
func NewRevokeAPIKey(storage Storage, audit AuditRecorder) *RevokeAPIKey {
	return &RevokeAPIKey{storage: storage, audit: audit}
}

// Execute deletes the API key with the given ID, scoped to the calling user.
// Revoking a key that does not exist is a no-op.
func (uc *RevokeAPIKey) Execute(ctx context.Context, id, userID uuid.UUID) error {
	apiKeyFilter := &APIKeyFilter{
		ID:     filter.Equals(id),
		UserID: filter.Equals(userID),
	}

	apiKey, err := uc.storage.APIKeys().First(ctx, apiKeyFilter)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return nil
		}

		return fmt.Errorf("fetching API key: %w", err)
	}

	if err := uc.storage.APIKeys().Delete(ctx, apiKeyFilter); err != nil {
		return fmt.Errorf("deleting API key: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   apiKey.WorkspaceID,
		ActorID:       userID,
		Action:        "delete",
		ResourceType:  "api_key",
		ResourceID:    apiKey.ID,
		ResourceLabel: apiKey.Name,
		Before:        map[string]any{"name": apiKey.Name},
	})

	return nil
}
