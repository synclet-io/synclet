package notifyservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// SlackChannel delivers notifications via Slack incoming webhooks.
type SlackChannel struct {
	secrets    SecretsProvider
	httpClient *http.Client
}

// NewSlackChannel creates a new SlackChannel deliverer.
func NewSlackChannel(secrets SecretsProvider) *SlackChannel {
	return &SlackChannel{
		secrets: secrets,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Deliver sends a notification to a Slack webhook URL.
func (s *SlackChannel) Deliver(ctx context.Context, channel *NotificationChannel, event WebhookEvent) error {
	var config map[string]string
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("parsing channel config: %w", err)
	}

	webhookURL, ok := config["webhook_url"]
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url missing from channel config")
	}

	// Decrypt webhook URL if it's a secret reference.
	if secretutil.IsSecretRef(webhookURL) {
		decrypted, err := s.secrets.RetrieveSecret(ctx, webhookURL)
		if err != nil {
			return fmt.Errorf("decrypting slack webhook_url: %w", err)
		}
		webhookURL = decrypted
	}

	payload := map[string]string{
		"text": formatSlackMessage(event),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending slack notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func formatSlackMessage(event WebhookEvent) string {
	msg := fmt.Sprintf("[Synclet] Connection sync %s", event.Event)
	if event.ConnectionID != "" {
		msg += fmt.Sprintf(" (connection: %s)", event.ConnectionID)
	}
	if event.Error != "" {
		msg += fmt.Sprintf(": %s", event.Error)
	}
	return msg
}
