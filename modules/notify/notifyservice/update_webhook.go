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

// UpdateWebhookParams holds parameters for updating a webhook. Nil fields stay
// unchanged; a non-nil empty Events unsubscribes from every event and an empty
// Secret stops signing deliveries.
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
	audit   AuditRecorder
}

// NewUpdateWebhook creates a new UpdateWebhook use case.
func NewUpdateWebhook(storage Storage, secrets SecretsProvider, audit AuditRecorder) *UpdateWebhook {
	return &UpdateWebhook{storage: storage, secrets: secrets, audit: audit}
}

// Execute updates the webhook matching the given ID and workspace.
func (uc *UpdateWebhook) Execute(ctx context.Context, params UpdateWebhookParams) (*Webhook, error) {
	webhook, err := uc.storage.Webhooks().First(ctx, &WebhookFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting webhook: %w", err)
	}

	before := map[string]any{
		"url":     webhook.URL,
		"enabled": webhook.Enabled,
		"events":  webhook.Events,
		"secret":  webhook.Secret,
	}

	if params.URL != nil {
		webhook.URL = *params.URL
	}

	if params.Events != nil {
		eventsJSON, err := json.Marshal(params.Events)
		if err != nil {
			return nil, fmt.Errorf("marshaling events: %w", err)
		}

		webhook.Events = string(eventsJSON)
	}

	var staleSecretRef string

	if params.Secret != nil {
		if secretutil.IsSecretRef(webhook.Secret) {
			staleSecretRef = webhook.Secret
		}

		webhook.Secret = ""

		if *params.Secret != "" {
			ref, err := uc.secrets.StoreSecret(ctx, "webhook", webhook.ID, *params.Secret)
			if err != nil {
				return nil, fmt.Errorf("encrypting webhook secret: %w", err)
			}

			webhook.Secret = ref
		}
	}

	if params.Enabled != nil {
		webhook.Enabled = *params.Enabled
	}

	webhook.UpdatedAt = time.Now()

	updated, err := uc.storage.Webhooks().Update(ctx, webhook)
	if err != nil {
		return nil, fmt.Errorf("updating webhook: %w", err)
	}

	if staleSecretRef != "" {
		_ = uc.secrets.DeleteSecret(ctx, staleSecretRef) // non-fatal
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "webhook",
		ResourceID:    updated.ID,
		ResourceLabel: updated.URL,
		Before:        before,
		After: map[string]any{
			"url":     updated.URL,
			"enabled": updated.Enabled,
			"events":  updated.Events,
			"secret":  updated.Secret,
		},
	})

	return updated, nil
}
