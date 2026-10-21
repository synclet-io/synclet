package authservice

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAPIKey_RecordsAuditEventWithoutSecrets(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	audit := &capturingAuditRecorder{}
	wsID, userID := uuid.New(), uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	rawKey, created, err := NewCreateAPIKey(store, audit).Execute(ctx, CreateAPIKeyParams{WorkspaceID: wsID, UserID: userID, Name: "ci", ExpiresAt: &expiresAt})
	require.NoError(t, err)

	require.Len(t, audit.events, 1)
	event := audit.events[0]
	assert.Equal(t, wsID, event.WorkspaceID)
	assert.Equal(t, userID, event.ActorID)
	assert.Equal(t, "create", event.Action)
	assert.Equal(t, "api_key", event.ResourceType)
	assert.Equal(t, created.ID, event.ResourceID)
	assert.Equal(t, "ci", event.ResourceLabel)
	assert.Nil(t, event.Before)
	assert.Equal(t, map[string]any{"name": "ci", "expires_at": &expiresAt}, event.After)

	payload, err := json.Marshal(event)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), rawKey)
	assert.NotContains(t, string(payload), created.KeyHash)
}

func TestCreateAPIKey_RejectsExpiryInThePast(t *testing.T) {
	store := newFakeStorage()
	expiresAt := time.Now().Add(-time.Minute)

	_, _, err := NewCreateAPIKey(store, &capturingAuditRecorder{}).Execute(context.Background(), CreateAPIKeyParams{WorkspaceID: uuid.New(), UserID: uuid.New(), Name: "ci", ExpiresAt: &expiresAt})

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Empty(t, store.apiKeys)
}
