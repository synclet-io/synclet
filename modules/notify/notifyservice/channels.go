package notifyservice

import "context"

// ChannelDeliverer delivers notifications to a specific channel type.
type ChannelDeliverer interface {
	Deliver(ctx context.Context, channel *NotificationChannel, event WebhookEvent) error
}
