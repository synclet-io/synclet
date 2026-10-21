package pipelineeventhandle

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

type workflowAdvancer interface {
	ExecuteForJob(ctx context.Context, job *pipelineservice.Job) error
}

// WorkflowJobHandler advances workflow runs when a job.updated event moves a
// job to completed, failed or cancelled: the run of the step the job belongs
// to, and runs with a step waiting on the job's connection. Advancing is
// idempotent, so redelivered events are harmless.
type WorkflowJobHandler struct {
	logger  *logging.Logger
	advance workflowAdvancer
}

var (
	_ pnpwatermill.Handler = (*WorkflowJobHandler)(nil)
	_ workflowAdvancer     = (*pipelinejobs.AdvanceWorkflowRuns)(nil)
)

// NewWorkflowJobHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the pipeline FX module.
func NewWorkflowJobHandler(logger *logging.Logger, advance *pipelinejobs.AdvanceWorkflowRuns) *WorkflowJobHandler {
	return &WorkflowJobHandler{
		logger:  logger.Named("workflow-job-handler"),
		advance: advance,
	}
}

func (h *WorkflowJobHandler) Name() string {
	return "pipeline_workflow_job"
}

func (h *WorkflowJobHandler) Topic() string {
	return pipelineevent.PipelineEventTopic
}

// Handle advances the runs a finished job can move. Undecodable messages are
// acked; advance failures are returned so the bus retries the delivery.
func (h *WorkflowJobHandler) Handle(msg *message.Message) error {
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

	if err := h.advance.ExecuteForJob(ctx, updated.NewData); err != nil {
		return fmt.Errorf("advancing workflow runs after job %s: %w", updated.NewData.ID, err)
	}

	return nil
}
