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
	storage pipelineservice.Storage
}

// NewBatchAppendJobLogs creates a new BatchAppendJobLogs use case.
func NewBatchAppendJobLogs(storage pipelineservice.Storage) *BatchAppendJobLogs {
	return &BatchAppendJobLogs{storage: storage}
}

// Execute appends the log lines to the job's log storage.
func (uc *BatchAppendJobLogs) Execute(ctx context.Context, params BatchAppendJobLogsParams) error {
	return uc.storage.JobLogs().BatchAppendLogs(ctx, params.JobID, params.LogLines)
}
