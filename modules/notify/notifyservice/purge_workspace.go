package notifyservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// PurgeWorkspace removes every notification record owned by a deleted workspace.
type PurgeWorkspace struct {
	storage Storage
	secrets SecretsProvider
}

// NewPurgeWorkspace creates a new PurgeWorkspace use case.
func NewPurgeWorkspace(storage Storage, secrets SecretsProvider) *PurgeWorkspace {
	return &PurgeWorkspace{storage: storage, secrets: secrets}
}

// Execute deletes the workspace's notification rules, channels and webhooks.
// Secrets are deleted before the rows that own them, so a failed step leaves
// the owners in place and a rerun finishes the job. Running it again on a
// purged workspace is a no-op.
func (uc *PurgeWorkspace) Execute(ctx context.Context, workspaceID uuid.UUID) error {
	if err := uc.storage.NotificationRules().Delete(ctx, &NotificationRuleFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting notification rules: %w", err)
	}

	channels, err := uc.storage.NotificationChannels().Find(ctx, &NotificationChannelFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing notification channels: %w", err)
	}

	for _, channel := range channels {
		if err := uc.secrets.DeleteByOwner(ctx, "channel", channel.ID); err != nil {
			return fmt.Errorf("deleting secrets of channel %s: %w", channel.ID, err)
		}
	}

	if err := uc.storage.NotificationChannels().Delete(ctx, &NotificationChannelFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting notification channels: %w", err)
	}

	webhooks, err := uc.storage.Webhooks().Find(ctx, &WebhookFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing webhooks: %w", err)
	}

	for _, webhook := range webhooks {
		if err := uc.secrets.DeleteByOwner(ctx, "webhook", webhook.ID); err != nil {
			return fmt.Errorf("deleting secrets of webhook %s: %w", webhook.ID, err)
		}
	}

	if err := uc.storage.Webhooks().Delete(ctx, &WebhookFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting webhooks: %w", err)
	}

	return nil
}
