package notifyservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// DeleteChannelParams holds parameters for deleting a notification channel.
type DeleteChannelParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// DeleteChannel deletes a notification channel and its associated rules.
type DeleteChannel struct {
	storage Storage
	secrets SecretsProvider
	audit   AuditRecorder
}

// NewDeleteChannel creates a new DeleteChannel use case.
func NewDeleteChannel(storage Storage, secrets SecretsProvider, audit AuditRecorder) *DeleteChannel {
	return &DeleteChannel{storage: storage, secrets: secrets, audit: audit}
}

// Execute deletes the notification channel and its rules. It returns a
// not-found error when no such channel exists.
func (uc *DeleteChannel) Execute(ctx context.Context, params DeleteChannelParams) error {
	existing, err := uc.storage.NotificationChannels().First(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting notification channel: %w", err)
	}

	if err := uc.storage.NotificationRules().Delete(ctx, &NotificationRuleFilter{
		ChannelID:   filter.Equals(existing.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return fmt.Errorf("deleting associated notification rules: %w", err)
	}

	if err := uc.storage.NotificationChannels().Delete(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(existing.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return fmt.Errorf("deleting notification channel: %w", err)
	}

	_ = uc.secrets.DeleteByOwner(ctx, "channel", existing.ID) // non-fatal

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "delete",
		ResourceType:  "notification_channel",
		ResourceID:    existing.ID,
		ResourceLabel: existing.Name,
		Before: map[string]any{
			"name":         existing.Name,
			"channel_type": existing.ChannelType.String(),
		},
	})

	return nil
}
