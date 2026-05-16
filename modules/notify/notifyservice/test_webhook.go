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

// TestWebhookParams holds parameters for testing a webhook.
type TestWebhookParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// TestWebhookResult is the outcome of a synthetic delivery attempt.
type TestWebhookResult struct {
	StatusCode    int
	DeliveryError string
}

// TestWebhook delivers a synthetic event to an existing webhook so operators
// can verify endpoint reachability and HMAC signature verification.
type TestWebhook struct {
	storage    Storage
	secrets    SecretsProvider
	httpClient *http.Client
}

// NewTestWebhook creates a new TestWebhook use case.
func NewTestWebhook(storage Storage, secrets SecretsProvider, cfg Config) *TestWebhook {
	return &TestWebhook{
		storage:    storage,
		secrets:    secrets,
		httpClient: &http.Client{Timeout: cfg.WebhookHTTPTimeout},
	}
}

// Execute sends a webhook.test event to the webhook and returns the HTTP
// status. A non-2xx response is reported via DeliveryError, not as an error.
func (uc *TestWebhook) Execute(ctx context.Context, params TestWebhookParams) (TestWebhookResult, error) {
	webhook, err := uc.storage.Webhooks().First(ctx, &WebhookFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return TestWebhookResult{}, fmt.Errorf("loading webhook: %w", err)
	}

	if err := connectutil.ValidateWebhookURLAtDelivery(webhook.URL); err != nil {
		return TestWebhookResult{}, fmt.Errorf("validating url: %w", err)
	}

	payload, err := json.Marshal(WebhookEvent{
		Event:     "webhook.test",
		Timestamp: time.Now(),
	})
	if err != nil {
		return TestWebhookResult{}, fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return TestWebhookResult{}, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if webhook.Secret != "" {
		secret := webhook.Secret
		if secretutil.IsSecretRef(secret) {
			decrypted, err := uc.secrets.RetrieveSecret(ctx, secret)
			if err != nil {
				return TestWebhookResult{}, fmt.Errorf("decrypting webhook secret: %w", err)
			}

			secret = decrypted
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		req.Header.Set("X-Synclet-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return TestWebhookResult{StatusCode: 0, DeliveryError: err.Error()}, nil
	}

	defer func() { _ = resp.Body.Close() }()

	result := TestWebhookResult{StatusCode: resp.StatusCode}
	if resp.StatusCode >= 300 {
		result.DeliveryError = fmt.Sprintf("endpoint returned status %d", resp.StatusCode)
	}

	return result, nil
}
