package notifyservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// TestChannelParams holds parameters for testing a notification channel.
type TestChannelParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// TestChannel sends a test notification to verify channel configuration.
type TestChannel struct {
	storage    Storage
	deliverers map[ChannelType]ChannelDeliverer
}

// NewTestChannel creates a new TestChannel use case.
func NewTestChannel(storage Storage, deliverers map[ChannelType]ChannelDeliverer) *TestChannel {
	return &TestChannel{
		storage:    storage,
		deliverers: deliverers,
	}
}

// Execute sends a test message to the specified channel.
func (uc *TestChannel) Execute(ctx context.Context, params TestChannelParams) error {
	channel, err := uc.storage.NotificationChannels().First(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting notification channel: %w", err)
	}

	deliverer, ok := uc.deliverers[channel.ChannelType]
	if !ok {
		return fmt.Errorf("no deliverer for channel type %s", channel.ChannelType)
	}

	testEvent := WebhookEvent{
		Event:     "test",
		Timestamp: time.Now(),
	}

	if err := deliverer.Deliver(ctx, channel, testEvent); err != nil {
		return fmt.Errorf("test delivery failed: %w", err)
	}

	return nil
}
