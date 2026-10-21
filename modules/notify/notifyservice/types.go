package notifyservice

import "time"

// jsonb is the type used for JSONB columns in PostgreSQL.
type jsonb = string

// WebhookEvent represents a webhook payload.
type WebhookEvent struct {
	Event        string    `json:"event"`
	Timestamp    time.Time `json:"timestamp"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	ConnectionID string    `json:"connection_id,omitempty"`
	// ConnectionName is the human-readable connection name (best-effort enrichment).
	ConnectionName string `json:"connection_name,omitempty"`
	// SourceName / DestinationName are the human-readable endpoint names (best-effort enrichment).
	SourceName      string `json:"source_name,omitempty"`
	DestinationName string `json:"destination_name,omitempty"`
	JobID           string `json:"job_id,omitempty"`
	JobType         string `json:"job_type,omitempty"`
	// Attempt is the attempt number that triggered this event (1-based).
	Attempt int `json:"attempt,omitempty"`
	// RecordsRead, BytesSynced and DurationMs come from the job's latest attempt when available.
	RecordsRead int64 `json:"records_read,omitempty"`
	BytesSynced int64 `json:"bytes_synced,omitempty"`
	DurationMs  int64 `json:"duration_ms,omitempty"`
	// WorkflowID, WorkflowName and WorkflowRunID identify the workflow run of a workflow event.
	WorkflowID    string `json:"workflow_id,omitempty"`
	WorkflowName  string `json:"workflow_name,omitempty"`
	WorkflowRunID string `json:"workflow_run_id,omitempty"`
	Error         string `json:"error,omitempty"`
}
