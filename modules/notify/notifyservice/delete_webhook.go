package notifyservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// DeleteWebhookParams holds parameters for deleting a webhook.
type DeleteWebhookParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// DeleteWebhook deletes a webhook by ID and workspace.
type DeleteWebhook struct {
	storage Storage
	secrets SecretsProvider
}

// NewDeleteWebhook creates a new DeleteWebhook use case.
func NewDeleteWebhook(storage Storage, secrets SecretsProvider) *DeleteWebhook {
	return &DeleteWebhook{storage: storage, secrets: secrets}
}

// Execute deletes the webhook matching the given ID and workspace.
func (uc *DeleteWebhook) Execute(ctx context.Context, params DeleteWebhookParams) error {
	// Load webhook to check if secret needs cleanup.
	webhook, err := uc.storage.Webhooks().First(ctx, &WebhookFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting webhook: %w", err)
	}

	// Clean up encrypted secret if present.
	if webhook.Secret != "" && secretutil.IsSecretRef(webhook.Secret) {
		_ = uc.secrets.DeleteSecret(ctx, webhook.Secret) // non-fatal
	}

	return uc.storage.Webhooks().Delete(ctx, &WebhookFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
}
