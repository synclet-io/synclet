package notifyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// UpdateWebhookParams holds parameters for updating a webhook.
type UpdateWebhookParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	URL         *string
	Events      []string
	Secret      *string
	Enabled     *bool
}

// UpdateWebhook updates an existing webhook.
type UpdateWebhook struct {
	storage Storage
	secrets SecretsProvider
}

// NewUpdateWebhook creates a new UpdateWebhook use case.
func NewUpdateWebhook(storage Storage, secrets SecretsProvider) *UpdateWebhook {
	return &UpdateWebhook{storage: storage, secrets: secrets}
}

// Execute updates the webhook matching the given ID and workspace.
func (uc *UpdateWebhook) Execute(ctx context.Context, params UpdateWebhookParams) (*Webhook, error) {
	wh, err := uc.storage.Webhooks().First(ctx, &WebhookFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting webhook: %w", err)
	}

	if params.URL != nil {
		wh.URL = *params.URL
	}
	if params.Events != nil {
		eventsJSON, err := json.Marshal(params.Events)
		if err != nil {
			return nil, fmt.Errorf("marshaling events: %w", err)
		}
		wh.Events = string(eventsJSON)
	}
	if params.Secret != nil && *params.Secret != "" && !secretutil.IsSecretRef(*params.Secret) {
		// Delete old secret if it was encrypted.
		if secretutil.IsSecretRef(wh.Secret) {
			_ = uc.secrets.DeleteSecret(ctx, wh.Secret) // non-fatal
		}
		ref, err := uc.secrets.StoreSecret(ctx, "webhook", wh.ID, *params.Secret)
		if err != nil {
			return nil, fmt.Errorf("encrypting webhook secret: %w", err)
		}
		wh.Secret = ref
	}
	if params.Enabled != nil {
		wh.Enabled = *params.Enabled
	}

	wh.UpdatedAt = time.Now()

	updated, err := uc.storage.Webhooks().Update(ctx, wh)
	if err != nil {
		return nil, fmt.Errorf("updating webhook: %w", err)
	}

	return updated, nil
}
