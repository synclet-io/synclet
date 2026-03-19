package notifyservice

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopSecretsProvider is a test double that passes through values unchanged.
type noopSecretsProvider struct{}

func (n *noopSecretsProvider) StoreSecret(_ context.Context, _ string, _ uuid.UUID, plaintext string) (string, error) {
	return plaintext, nil
}
func (n *noopSecretsProvider) RetrieveSecret(_ context.Context, secretRef string) (string, error) {
	return secretRef, nil
}
func (n *noopSecretsProvider) DeleteSecret(_ context.Context, _ string) error { return nil }
func (n *noopSecretsProvider) DeleteByOwner(_ context.Context, _ string, _ uuid.UUID) error {
	return nil
}

func TestSlackChannel_Deliver_SendsCorrectPayload(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	slack := NewSlackChannel(&noopSecretsProvider{})
	channel := &NotificationChannel{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Name:        "test-slack",
		ChannelType: ChannelTypeSlack,
		Config:      mustJSON(map[string]string{"webhook_url": server.URL}),
		Enabled:     true,
	}

	event := WebhookEvent{
		Event:        "sync.failed",
		Timestamp:    time.Now(),
		ConnectionID: "conn-123",
		Error:        "timeout exceeded",
	}

	err := slack.Deliver(context.Background(), channel, event)
	require.NoError(t, err)

	assert.Equal(t, "application/json", receivedContentType)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(receivedBody, &payload))
	assert.Contains(t, payload["text"], "[Synclet]")
	assert.Contains(t, payload["text"], "sync.failed")
	assert.Contains(t, payload["text"], "conn-123")
	assert.Contains(t, payload["text"], "timeout exceeded")
}

func TestSlackChannel_Deliver_ReturnsErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	slack := NewSlackChannel(&noopSecretsProvider{})
	channel := &NotificationChannel{
		ID:          uuid.New(),
		ChannelType: ChannelTypeSlack,
		Config:      mustJSON(map[string]string{"webhook_url": server.URL}),
	}

	err := slack.Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestSlackChannel_Deliver_ReturnsErrorWhenWebhookURLMissing(t *testing.T) {
	slack := NewSlackChannel(&noopSecretsProvider{})
	channel := &NotificationChannel{
		ID:          uuid.New(),
		ChannelType: ChannelTypeSlack,
		Config:      mustJSON(map[string]string{}),
	}

	err := slack.Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook_url")
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
