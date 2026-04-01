package notifyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateWebhookParams holds parameters for creating a webhook.
type CreateWebhookParams struct {
	WorkspaceID uuid.UUID
	URL         string
	Events      []string
	Secret      string
}

// CreateWebhook creates a new webhook for a workspace.
type CreateWebhook struct {
	storage Storage
	secrets SecretsProvider
}

// NewCreateWebhook creates a new CreateWebhook use case.
func NewCreateWebhook(storage Storage, secrets SecretsProvider) *CreateWebhook {
	return &CreateWebhook{storage: storage, secrets: secrets}
}

// Execute creates a webhook with the given parameters.
func (uc *CreateWebhook) Execute(ctx context.Context, params CreateWebhookParams) (*Webhook, error) {
	now := time.Now()

	eventsJSON, err := json.Marshal(params.Events)
	if err != nil {
		return nil, fmt.Errorf("marshaling events: %w", err)
	}

	webhook := &Webhook{
		ID:          uuid.New(),
		WorkspaceID: params.WorkspaceID,
		URL:         params.URL,
		Events:      string(eventsJSON),
		Secret:      params.Secret,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Encrypt webhook secret if provided.
	if params.Secret != "" {
		ref, err := uc.secrets.StoreSecret(ctx, "webhook", webhook.ID, params.Secret)
		if err != nil {
			return nil, fmt.Errorf("encrypting webhook secret: %w", err)
		}

		webhook.Secret = ref
	}

	created, err := uc.storage.Webhooks().Create(ctx, webhook)
	if err != nil {
		return nil, fmt.Errorf("creating webhook: %w", err)
	}

	return created, nil
}
