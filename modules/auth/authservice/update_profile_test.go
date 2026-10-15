package authservice

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProfile_RecordsBeforeAndAfter(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	audit := &capturingAuditRecorder{}

	store := newFakeStorage()
	store.seedUser(&User{ID: userID, Email: "a@example.com", Name: "Old", PasswordHash: "hash-secret"})

	updated, err := NewUpdateProfile(store, audit).Execute(ctx, userID, "New")
	require.NoError(t, err)
	assert.Equal(t, "New", updated.Name)

	require.Len(t, audit.events, 1)
	event := audit.events[0]
	assert.Equal(t, uuid.Nil, event.WorkspaceID)
	assert.Equal(t, userID, event.ActorID)
	assert.Equal(t, "update", event.Action)
	assert.Equal(t, "user", event.ResourceType)
	assert.Equal(t, userID, event.ResourceID)
	assert.Equal(t, "a@example.com", event.ResourceLabel)
	assert.Equal(t, map[string]any{"name": "Old", "email": "a@example.com"}, event.Before)
	assert.Equal(t, map[string]any{"name": "New", "email": "a@example.com"}, event.After)

	payload, err := json.Marshal(event)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "hash-secret")
}

func TestUpdateProfile_NotFoundRecordsNothing(t *testing.T) {
	audit := &capturingAuditRecorder{}

	_, err := NewUpdateProfile(newFakeStorage(), audit).Execute(context.Background(), uuid.New(), "New")
	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Empty(t, audit.events)
}
