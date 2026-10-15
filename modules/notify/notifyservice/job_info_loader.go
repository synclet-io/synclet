package notifyservice

import (
	"context"

	"github.com/google/uuid"
)

// JobInfo holds contextual data the notify module needs to enrich webhook
// payloads for job-related events. It is loaded on demand by a JobInfoLoader
// implementation provided via the adapter layer.
type JobInfo struct {
	ConnectionName  string
	SourceName      string
	DestinationName string
	// LatestAttempt is the most recent JobAttempt for the job, if any.
	LatestAttempt *JobAttemptInfo
}

// JobAttemptInfo summarizes a single attempt for webhook payload enrichment.
type JobAttemptInfo struct {
	AttemptNumber int
	RecordsRead   int64
	BytesSynced   int64
	DurationMs    int64
}

// JobInfoLoader is the port the notify module uses to fetch job/connection
// details from the pipeline module. The implementation lives in
// modules/notify/notifyadapt and calls pipeline use cases. Notify never imports
// pipeline directly.
type JobInfoLoader interface {
	LoadJobInfo(ctx context.Context, workspaceID, connectionID, jobID uuid.UUID) (*JobInfo, error)
}
