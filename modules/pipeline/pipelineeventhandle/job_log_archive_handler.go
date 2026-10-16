package pipelineeventhandle

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
)

type jobLogArchiver interface {
	Execute(ctx context.Context, jobID uuid.UUID) error
}

// JobLogArchiveHandler archives a job's logs when a job.updated event moves
// the job to completed, failed or cancelled. Archiving is idempotent, so
// redelivered events are harmless.
type JobLogArchiveHandler struct {
	logger  *logging.Logger
	archive jobLogArchiver
}

var (
	_ pnpwatermill.Handler = (*JobLogArchiveHandler)(nil)
	_ jobLogArchiver       = (*pipelinelogs.ArchiveJobLogs)(nil)
)

// NewJobLogArchiveHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the pipeline FX module.
func NewJobLogArchiveHandler(logger *logging.Logger, archive *pipelinelogs.ArchiveJobLogs) *JobLogArchiveHandler {
	return &JobLogArchiveHandler{
		logger:  logger.Named("job-log-archive-handler"),
		archive: archive,
	}
}

func (h *JobLogArchiveHandler) Name() string {
	return "pipeline_job_log_archive"
}

func (h *JobLogArchiveHandler) Topic() string {
	return pipelineevent.PipelineEventTopic
}

// Handle archives the logs of a job that just finished. Undecodable messages
// are acked; archive failures are returned so the bus retries the delivery.
func (h *JobLogArchiveHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := pipelineevent.DecodePipelineEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode pipeline event; acking and dropping")

		return nil
	}

	updated, ok := event.Data.(*pipelineservice.JobUpdatedEventData)
	if !ok || updated == nil || updated.OldData == nil || updated.NewData == nil {
		return nil
	}

	if updated.OldData.Status == updated.NewData.Status || !isFinished(updated.NewData.Status) {
		return nil
	}

	if err := h.archive.Execute(ctx, updated.NewData.ID); err != nil {
		return fmt.Errorf("archiving logs of job %s: %w", updated.NewData.ID, err)
	}

	return nil
}

func isFinished(status pipelineservice.JobStatus) bool {
	return status == pipelineservice.JobStatusCompleted ||
		status == pipelineservice.JobStatusFailed ||
		status == pipelineservice.JobStatusCancelled
}
