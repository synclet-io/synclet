package pipelinelogs

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// AppendJobLogParams holds parameters for appending a log line.
type AppendJobLogParams struct {
	JobID   uuid.UUID
	LogLine string
}

// AppendJobLog appends a single log line to a job's log.
type AppendJobLog struct {
	storage pipelineservice.Storage
}

// NewAppendJobLog creates a new AppendJobLog use case.
func NewAppendJobLog(storage pipelineservice.Storage) *AppendJobLog {
	return &AppendJobLog{storage: storage}
}

// Execute appends the log line to the job's log storage.
func (uc *AppendJobLog) Execute(ctx context.Context, params AppendJobLogParams) error {
	return uc.storage.JobLogs().AppendLog(ctx, params.JobID, params.LogLine)
}
