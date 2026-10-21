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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

type runAdvancer interface {
	Execute(ctx context.Context, runIDs ...uuid.UUID) error
}

// WorkflowRunStartedHandler advances a workflow run when a workflow_run.started
// event reports that it was started or rerun from the API. Advancing is
// idempotent, so redelivered events are harmless.
type WorkflowRunStartedHandler struct {
	logger  *logging.Logger
	advance runAdvancer
}

var (
	_ pnpwatermill.Handler = (*WorkflowRunStartedHandler)(nil)
	_ runAdvancer          = (*pipelinejobs.AdvanceWorkflowRuns)(nil)
)

// NewWorkflowRunStartedHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the pipeline FX module.
func NewWorkflowRunStartedHandler(logger *logging.Logger, advance *pipelinejobs.AdvanceWorkflowRuns) *WorkflowRunStartedHandler {
	return &WorkflowRunStartedHandler{
		logger:  logger.Named("workflow-run-started-handler"),
		advance: advance,
	}
}

func (h *WorkflowRunStartedHandler) Name() string {
	return "pipeline_workflow_run_started"
}

func (h *WorkflowRunStartedHandler) Topic() string {
	return pipelineevent.PipelineEventTopic
}

// Handle enqueues the run's ready steps. Undecodable messages are acked;
// advance failures are returned so the bus retries the delivery.
func (h *WorkflowRunStartedHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := pipelineevent.DecodePipelineEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode pipeline event; acking and dropping")

		return nil
	}

	started, ok := event.Data.(*pipelineservice.WorkflowRunStartedEventData)
	if !ok || started == nil {
		return nil
	}

	if err := h.advance.Execute(ctx, started.RunID); err != nil {
		return fmt.Errorf("advancing workflow run %s: %w", started.RunID, err)
	}

	return nil
}
