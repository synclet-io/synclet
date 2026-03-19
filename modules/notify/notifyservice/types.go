package notifyservice

import "time"

// jsonb is the type used for JSONB columns in PostgreSQL.
type jsonb = string

// WebhookEvent represents a webhook payload.
type WebhookEvent struct {
	Event        string    `json:"event"`
	Timestamp    time.Time `json:"timestamp"`
	ConnectionID string    `json:"connection_id,omitempty"`
	JobID        string    `json:"job_id,omitempty"`
	Error        string    `json:"error,omitempty"`
}
