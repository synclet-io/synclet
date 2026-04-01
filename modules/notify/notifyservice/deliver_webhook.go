package notifyservice

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/connectutil"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// DeliverWebhookParams holds parameters for delivering a webhook event.
type DeliverWebhookParams struct {
	WorkspaceID uuid.UUID
	Event       WebhookEvent
}

// DeliverWebhook sends an event to all matching webhooks for a workspace.
type DeliverWebhook struct {
	storage    Storage
	secrets    SecretsProvider
	httpClient *http.Client
}

// NewDeliverWebhook creates a new DeliverWebhook use case.
func NewDeliverWebhook(storage Storage, secrets SecretsProvider) *DeliverWebhook {
	return &DeliverWebhook{
		storage: storage,
		secrets: secrets,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Execute delivers the event to all matching enabled webhooks for the workspace.
// Retries up to 3 times per webhook with linear backoff.
func (uc *DeliverWebhook) Execute(ctx context.Context, params DeliverWebhookParams) error {
	webhooks, err := uc.storage.Webhooks().Find(ctx, &WebhookFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
		Enabled:     filter.Equals(true),
	})
	if err != nil {
		return fmt.Errorf("listing webhooks: %w", err)
	}

	payload, err := json.Marshal(params.Event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	for _, webhook := range webhooks {
		if !webhookMatchesEvent(webhook, params.Event.Event) {
			continue
		}

		// Re-validate URL at delivery time to prevent DNS rebinding attacks.
		if err := connectutil.ValidateWebhookURLAtDelivery(webhook.URL); err != nil {
			continue
		}

		// Retry up to 3 times with context-aware backoff.
		for attempt := range 3 {
			if err := uc.sendWebhook(ctx, webhook, payload); err == nil {
				break
			}

			backoff := time.Duration(attempt+1) * time.Second
			select {
			case <-time.After(backoff):
				// Continue retry.
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

func (uc *DeliverWebhook) sendWebhook(ctx context.Context, webhook *Webhook, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Sign with HMAC-SHA256, decrypting secret if needed.
	if webhook.Secret != "" {
		secret := webhook.Secret
		if secretutil.IsSecretRef(secret) {
			decrypted, err := uc.secrets.RetrieveSecret(ctx, secret)
			if err != nil {
				return fmt.Errorf("decrypting webhook secret: %w", err)
			}

			secret = decrypted
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Synclet-Signature", sig)
	}

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func webhookMatchesEvent(webhook *Webhook, eventType string) bool {
	var events []string
	if err := json.Unmarshal([]byte(webhook.Events), &events); err != nil {
		return false
	}

	for _, e := range events {
		if e == eventType || e == "*" {
			return true
		}
	}

	return false
}
