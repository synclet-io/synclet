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
	audit   AuditRecorder
}

// NewCreateAPIKey creates a new CreateAPIKey use case.
func NewCreateAPIKey(storage Storage, audit AuditRecorder) *CreateAPIKey {
	return &CreateAPIKey{storage: storage, audit: audit}
}

// CreateAPIKeyParams describes an API key to create. OAuthGrantID and
// OAuthLevel are set on a key issued as an OAuth access token.
type CreateAPIKeyParams struct {
	WorkspaceID  uuid.UUID
	UserID       uuid.UUID
	Name         string
	ExpiresAt    *time.Time
	OAuthGrantID *uuid.UUID
	OAuthLevel   *OAuthAccessLevel
}

// Execute generates a new API key, stores its hash, and returns the raw key and record.
func (uc *CreateAPIKey) Execute(ctx context.Context, params CreateAPIKeyParams) (string, *APIKey, error) {
	rawKey, created, err := insertAPIKey(ctx, uc.storage, params)
	if err != nil {
		return "", nil, err
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   created.WorkspaceID,
		ActorID:       params.UserID,
		Action:        "create",
		ResourceType:  "api_key",
		ResourceID:    created.ID,
		ResourceLabel: created.Name,
		After:         map[string]any{"name": created.Name, "expires_at": created.ExpiresAt},
	})

	return rawKey, created, nil
}

func insertAPIKey(ctx context.Context, storage Storage, params CreateAPIKeyParams) (string, *APIKey, error) {
	if params.ExpiresAt != nil && !params.ExpiresAt.After(time.Now()) {
		return "", nil, &ValidationError{Message: "expiry must be in the future"}
	}

	rawKey, err := generateAPIKey()
	if err != nil {
		return "", nil, fmt.Errorf("generating key: %w", err)
	}

	created, err := storage.APIKeys().Create(ctx, &APIKey{
		ID:           uuid.New(),
		WorkspaceID:  params.WorkspaceID,
		UserID:       params.UserID,
		Name:         params.Name,
		KeyHash:      hashToken(rawKey),
		ExpiresAt:    params.ExpiresAt,
		OAuthGrantID: params.OAuthGrantID,
		OAuthLevel:   params.OAuthLevel,
		CreatedAt:    time.Now(),
	})
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
