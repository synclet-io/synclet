package authservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateAPIKey creates a new API key for a workspace.
type CreateAPIKey struct {
	storage Storage
}

// NewCreateAPIKey creates a new CreateAPIKey use case.
func NewCreateAPIKey(storage Storage) *CreateAPIKey {
	return &CreateAPIKey{storage: storage}
}

// Execute generates a new API key, stores its hash, and returns the raw key and record.
func (uc *CreateAPIKey) Execute(ctx context.Context, workspaceID, userID uuid.UUID, name string, expiresAt *time.Time) (string, *APIKey, error) {
	rawKey, err := generateAPIKey()
	if err != nil {
		return "", nil, fmt.Errorf("generating key: %w", err)
	}

	keyHash := hashToken(rawKey)

	apiKey := &APIKey{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Name:        name,
		KeyHash:     keyHash,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	created, err := uc.storage.APIKeys().Create(ctx, apiKey)
	if err != nil {
		return "", nil, fmt.Errorf("creating API key: %w", err)
	}

	return rawKey, created, nil
}

// generateAPIKey generates a new API key with the synclet_sk_ prefix.
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "synclet_sk_" + hex.EncodeToString(b), nil
}
