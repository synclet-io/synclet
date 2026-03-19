package notifyservice

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// NotificationEvent represents a sync lifecycle event type.
type NotificationEvent byte

const (
	NotificationEventCompleted NotificationEvent = iota + 1
	NotificationEventFailed
)

func (e NotificationEvent) String() string {
	switch e {
	case NotificationEventCompleted:
		return "sync.completed"
	case NotificationEventFailed:
		return "sync.failed"
	default:
		return "unknown"
	}
}

// DeliverNotificationParams holds parameters for delivering notifications.
type DeliverNotificationParams struct {
	WorkspaceID         uuid.UUID
	ConnectionID        uuid.UUID
	Event               NotificationEvent
	ConsecutiveFailures int
	RecordsSynced       int64
}

// DeliverNotification evaluates notification rules and dispatches to matching channels.
type DeliverNotification struct {
	storage    Storage
	deliverers map[ChannelType]ChannelDeliverer
	logger     *logging.Logger
}

// NewDeliverNotification creates a new DeliverNotification use case.
func NewDeliverNotification(storage Storage, deliverers map[ChannelType]ChannelDeliverer, logger *logging.Logger) *DeliverNotification {
	return &DeliverNotification{
		storage:    storage,
		deliverers: deliverers,
		logger:     logger,
	}
}

// Execute evaluates all applicable rules and dispatches notifications to matching channels.
// Notification delivery is best-effort: errors are logged but do not cause the operation to fail.
func (uc *DeliverNotification) Execute(ctx context.Context, params DeliverNotificationParams) error {
	// Load all enabled rules for this workspace that either match the specific connection
	// or are workspace-level defaults (ConnectionID is zero UUID).
	rules, err := uc.storage.NotificationRules().Find(ctx, &NotificationRuleFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
		Enabled:     filter.Equals(true),
		Or: []*NotificationRuleFilter{
			{ConnectionID: filter.Equals(params.ConnectionID)},
			{ConnectionID: filter.Equals(uuid.Nil)},
		},
	})
	if err != nil {
		return fmt.Errorf("loading notification rules: %w", err)
	}

	// Track which channels have been dispatched to via connection-specific rules
	// to avoid duplicate notifications from workspace-level defaults.
	dispatchedChannels := make(map[uuid.UUID]bool)

	// Process connection-specific rules first.
	for _, rule := range rules {
		if rule.ConnectionID == uuid.Nil {
			continue
		}

		if !evaluateCondition(rule, params) {
			continue
		}

		dispatchedChannels[rule.ChannelID] = true
		uc.dispatchToChannel(ctx, rule.ChannelID, params.WorkspaceID, params)
	}

	// Process workspace-level defaults, skipping channels already dispatched.
	for _, rule := range rules {
		if rule.ConnectionID != uuid.Nil {
			continue
		}

		if dispatchedChannels[rule.ChannelID] {
			continue
		}

		if !evaluateCondition(rule, params) {
			continue
		}

		uc.dispatchToChannel(ctx, rule.ChannelID, params.WorkspaceID, params)
	}

	return nil
}

func evaluateCondition(rule *NotificationRule, params DeliverNotificationParams) bool {
	switch rule.Condition {
	case NotificationConditionOnFailure:
		return params.Event == NotificationEventFailed
	case NotificationConditionOnConsecutiveFailures:
		return params.Event == NotificationEventFailed && params.ConsecutiveFailures >= rule.ConditionValue
	case NotificationConditionOnZeroRecords:
		return params.Event == NotificationEventCompleted && params.RecordsSynced == 0
	default:
		return false
	}
}

func (uc *DeliverNotification) dispatchToChannel(ctx context.Context, channelID, workspaceID uuid.UUID, params DeliverNotificationParams) {
	channel, err := uc.storage.NotificationChannels().First(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(channelID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		uc.logger.WithError(err).WithField("channel_id", channelID).Error(ctx, "loading notification channel")
		return
	}

	if !channel.Enabled {
		return
	}

	deliverer, ok := uc.deliverers[channel.ChannelType]
	if !ok {
		uc.logger.WithField("channel_type", channel.ChannelType).Warn(ctx, "no deliverer for channel type")
		return
	}

	event := WebhookEvent{
		Event:        params.Event.String(),
		Timestamp:    time.Now(),
		ConnectionID: params.ConnectionID.String(),
	}

	if err := deliverer.Deliver(ctx, channel, event); err != nil {
		uc.logger.WithError(err).WithFields(map[string]interface{}{"channel_id": channelID, "channel_type": channel.ChannelType}).Error(ctx, "delivering notification")
	}
}
