package pipelinelogs

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// BatchAppendJobLogsParams holds parameters for batch appending log lines.
type BatchAppendJobLogsParams struct {
	JobID    uuid.UUID
	LogLines []string
}

// BatchAppendJobLogs appends multiple log lines to a job's log in one call.
type BatchAppendJobLogs struct {
	storage  pipelineservice.Storage
	maxLines int
}

// NewBatchAppendJobLogs creates a new BatchAppendJobLogs use case.
func NewBatchAppendJobLogs(storage pipelineservice.Storage, config pipelineservice.Config) *BatchAppendJobLogs {
	return &BatchAppendJobLogs{storage: storage, maxLines: config.MaxLogLinesPerAttempt}
}

// Execute appends the log lines to the job's current attempt, subject to the
// per-attempt line cap.
func (uc *BatchAppendJobLogs) Execute(ctx context.Context, params BatchAppendJobLogsParams) error {
	return appendLines(ctx, uc.storage, uc.maxLines, params.JobID, params.LogLines)
}
