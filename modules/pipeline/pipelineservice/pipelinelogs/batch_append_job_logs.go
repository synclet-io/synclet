package pipelinelogs

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
)

// BatchAppendJobLogsParams holds parameters for batch appending log lines.
type BatchAppendJobLogsParams struct {
	JobID    uuid.UUID
	LogLines []string
}

// BatchAppendJobLogs appends multiple log lines to a job's log in one call.
type BatchAppendJobLogs struct {
	storage  pipelineservice.Storage
	metrics  *pipelinemetrics.MetricsCollector
	maxLines int
}

// NewBatchAppendJobLogs creates a new BatchAppendJobLogs use case.
func NewBatchAppendJobLogs(storage pipelineservice.Storage, config pipelineservice.Config, metrics *pipelinemetrics.MetricsCollector) *BatchAppendJobLogs {
	return &BatchAppendJobLogs{storage: storage, metrics: metrics, maxLines: config.MaxLogLinesPerAttempt}
}

// Execute appends the log lines to the job's current attempt, subject to the
// per-attempt line cap.
func (uc *BatchAppendJobLogs) Execute(ctx context.Context, params BatchAppendJobLogsParams) error {
	return appendLines(ctx, uc.storage, uc.metrics, uc.maxLines, params.JobID, params.LogLines)
}
