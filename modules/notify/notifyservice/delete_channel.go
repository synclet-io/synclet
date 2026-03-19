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
}

// NewDeleteChannel creates a new DeleteChannel use case.
func NewDeleteChannel(storage Storage, secrets SecretsProvider) *DeleteChannel {
	return &DeleteChannel{storage: storage, secrets: secrets}
}

// Execute deletes the notification channel and its rules.
func (uc *DeleteChannel) Execute(ctx context.Context, params DeleteChannelParams) error {
	// Clean up all encrypted secrets for this channel.
	_ = uc.secrets.DeleteByOwner(ctx, "channel", params.ID) // non-fatal

	// Delete associated notification rules first.
	if err := uc.storage.NotificationRules().Delete(ctx, &NotificationRuleFilter{
		ChannelID:   filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return fmt.Errorf("deleting associated notification rules: %w", err)
	}

	return uc.storage.NotificationChannels().Delete(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
}
