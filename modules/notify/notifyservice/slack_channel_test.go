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

	"github.com/synclet-io/synclet/pkg/connectutil"
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

	slack := NewSlackChannelWithClient(&noopSecretsProvider{}, server.Client())
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

	slack := NewSlackChannelWithClient(&noopSecretsProvider{}, server.Client())
	channel := &NotificationChannel{
		ID:          uuid.New(),
		ChannelType: ChannelTypeSlack,
		Config:      mustJSON(map[string]string{"webhook_url": server.URL}),
	}

	err := slack.Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestSlackChannel_Deliver_ReturnsErrorWhenWebhookURLMissing(t *testing.T) {
	slack := NewSlackChannel(&noopSecretsProvider{}, Config{WebhookHTTPTimeout: 10 * time.Second})
	channel := &NotificationChannel{
		ID:          uuid.New(),
		ChannelType: ChannelTypeSlack,
		Config:      mustJSON(map[string]string{}),
	}

	err := slack.Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook_url")
}

func TestSlackChannel_Deliver_RefusesInternalAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	slack := NewSlackChannel(&noopSecretsProvider{}, Config{WebhookHTTPTimeout: 10 * time.Second})
	channel := &NotificationChannel{
		ID:          uuid.New(),
		ChannelType: ChannelTypeSlack,
		Config:      mustJSON(map[string]string{"webhook_url": server.URL}),
	}

	err := slack.Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"})
	require.ErrorIs(t, err, connectutil.ErrWebhookAddressBlocked)
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)

	return string(b)
}

func TestFormatMessages_WorkflowFailed(t *testing.T) {
	event := WebhookEvent{
		Event:         "workflow.failed",
		WorkflowID:    "wf-id",
		WorkflowName:  "nightly",
		WorkflowRunID: "run-id",
		Error:         "boom",
	}

	assert.Equal(t, "[Synclet] Workflow nightly run failed (workflow: wf-id, run: run-id): boom", formatSlackMessage(event))
	assert.Equal(t, "[Synclet] Workflow nightly run failed\nWorkflow: wf-id\nRun: run-id\nError: boom", formatTelegramMessage(event))
	assert.Contains(t, formatEmailBody(event), "<p><strong>Workflow:</strong> nightly (wf-id)</p>")
}

func TestFormatEmailBody_EscapesEventValues(t *testing.T) {
	const markup = `<a href="https://evil">x</a>`

	const escaped = `&lt;a href=&#34;https://evil&#34;&gt;x&lt;/a&gt;`

	for name, event := range map[string]WebhookEvent{
		"sync": {
			Event: markup, ConnectionID: markup, JobID: markup, Error: markup,
		},
		"workflow": {
			Event: markup, WorkflowID: markup, WorkflowName: markup, WorkflowRunID: markup, Error: markup,
		},
	} {
		t.Run(name, func(t *testing.T) {
			body := formatEmailBody(event)

			assert.NotContains(t, body, "<a ")
			assert.Contains(t, body, "<p><strong>Error:</strong> "+escaped+"</p>")
		})
	}
}
