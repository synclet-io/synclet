package notifyservice

import "time"

// Config holds notify service configuration.
type Config struct {
	WebhookHTTPTimeout time.Duration
	WebhookMaxRetries  int
}
