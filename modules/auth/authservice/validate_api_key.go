package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ValidateAPIKey validates an API key and returns the key record.
type ValidateAPIKey struct {
	storage Storage
	logger  *logging.Logger
}

// NewValidateAPIKey creates a new ValidateAPIKey use case.
func NewValidateAPIKey(storage Storage, logger *logging.Logger) *ValidateAPIKey {
	return &ValidateAPIKey{storage: storage, logger: logger}
}

// Execute validates the raw API key, checks expiry, and updates last used timestamp.
func (uc *ValidateAPIKey) Execute(ctx context.Context, rawKey string) (*APIKey, error) {
	keyHash := hashToken(rawKey)

	apiKey, err := uc.storage.APIKeys().First(ctx, &APIKeyFilter{
		KeyHash: filter.Equals(keyHash),
	})
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used.
	now := time.Now()
	apiKey.LastUsedAt = &now
	if _, err := uc.storage.APIKeys().Update(ctx, apiKey); err != nil {
		uc.logger.WithError(err).WithField("api_key_id", apiKey.ID).Warn(ctx, "failed to update API key last used timestamp")
	}

	return apiKey, nil
}
