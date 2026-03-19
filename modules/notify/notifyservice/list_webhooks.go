package notifyservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// ListWebhooksParams holds parameters for listing webhooks.
type ListWebhooksParams struct {
	WorkspaceID uuid.UUID
}

// ListWebhooks returns all webhooks for a workspace.
type ListWebhooks struct {
	storage Storage
}

// NewListWebhooks creates a new ListWebhooks use case.
func NewListWebhooks(storage Storage) *ListWebhooks {
	return &ListWebhooks{storage: storage}
}

// Execute returns all webhooks for the given workspace with secrets masked.
func (uc *ListWebhooks) Execute(ctx context.Context, params ListWebhooksParams) ([]*Webhook, error) {
	webhooks, err := uc.storage.Webhooks().Find(ctx, &WebhookFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, err
	}

	// Mask secret fields in API responses.
	for _, wh := range webhooks {
		if wh.Secret != "" {
			wh.Secret = secretutil.SecretMask
		}
	}

	return webhooks, nil
}
