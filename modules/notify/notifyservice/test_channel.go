package notifyservice

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
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
	audit      AuditRecorder
}

// NewTestChannel creates a new TestChannel use case.
func NewTestChannel(storage Storage, deliverers map[ChannelType]ChannelDeliverer, audit AuditRecorder) *TestChannel {
	return &TestChannel{
		storage:    storage,
		deliverers: deliverers,
		audit:      audit,
	}
}

// TestChannelResult reports whether the channel accepted the test message.
type TestChannelResult struct {
	Delivered     bool
	DeliveryError string
}

// Execute sends a test message to the specified channel. A delivery failure
// is reported in the result, not as an error.
func (uc *TestChannel) Execute(ctx context.Context, params TestChannelParams) (TestChannelResult, error) {
	channel, err := uc.storage.NotificationChannels().First(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return TestChannelResult{}, fmt.Errorf("getting notification channel: %w", err)
	}

	deliverer, ok := uc.deliverers[channel.ChannelType]
	if !ok {
		return TestChannelResult{}, fmt.Errorf("no deliverer for channel type %s", channel.ChannelType)
	}

	testEvent := WebhookEvent{
		Event:     "test",
		Timestamp: time.Now(),
	}

	result := TestChannelResult{Delivered: true}
	if deliverErr := deliverer.Deliver(ctx, channel, testEvent); deliverErr != nil {
		result = TestChannelResult{DeliveryError: deliveryErrorMessage(deliverErr)}
	}

	after := map[string]any{"delivered": result.Delivered}
	if !result.Delivered {
		after["delivery_error"] = result.DeliveryError
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "test",
		ResourceType:  "notification_channel",
		ResourceID:    channel.ID,
		ResourceLabel: channel.Name,
		After:         after,
	})

	return result, nil
}

// deliveryErrorMessage hides request URLs, which carry the Slack webhook path
// or the Telegram bot token.
func deliveryErrorMessage(err error) string {
	msg := err.Error()

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.URL != "" {
		msg = strings.ReplaceAll(msg, urlErr.URL, "[redacted]")
	}

	return msg
}
