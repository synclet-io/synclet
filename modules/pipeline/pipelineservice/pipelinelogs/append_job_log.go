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
	storage  pipelineservice.Storage
	maxLines int
}

// NewAppendJobLog creates a new AppendJobLog use case.
func NewAppendJobLog(storage pipelineservice.Storage, config pipelineservice.Config) *AppendJobLog {
	return &AppendJobLog{storage: storage, maxLines: config.MaxLogLinesPerAttempt}
}

// Execute appends the log line to the job's current attempt, subject to the
// per-attempt line cap.
func (uc *AppendJobLog) Execute(ctx context.Context, params AppendJobLogParams) error {
	return appendLines(ctx, uc.storage, uc.maxLines, params.JobID, []string{params.LogLine})
}
