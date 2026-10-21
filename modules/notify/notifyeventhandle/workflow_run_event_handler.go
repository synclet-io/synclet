package notifyeventhandle

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

const webhookEventWorkflowFailed = "workflow.failed"

// WorkflowRunEventHandler reacts to workflow_run.finished. A failed run
// sends the workflow.failed webhook event and notifies the channels whose
// failure rules cover a connection that is a step of the workflow, or the
// whole workspace.
type WorkflowRunEventHandler struct {
	logger              *logging.Logger
	deliverWebhook      webhookDeliverer
	deliverNotification notificationDeliverer
}

var _ pnpwatermill.Handler = (*WorkflowRunEventHandler)(nil)

// NewWorkflowRunEventHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the notify FX module.
func NewWorkflowRunEventHandler(
	logger *logging.Logger,
	deliverWebhook *notifyservice.DeliverWebhook,
	deliverNotification *notifyservice.DeliverNotification,
) *WorkflowRunEventHandler {
	return &WorkflowRunEventHandler{
		logger:              logger.Named("workflow-run-event-handler"),
		deliverWebhook:      deliverWebhook,
		deliverNotification: deliverNotification,
	}
}

func (h *WorkflowRunEventHandler) Name() string {
	return "notify_pipeline_workflow_run_event"
}

func (h *WorkflowRunEventHandler) Topic() string {
	return pipelineevent.PipelineEventTopic
}

// Handle delivers the failure of a workflow run. Decode and delivery
// failures are logged and the message is acked.
func (h *WorkflowRunEventHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := pipelineevent.DecodePipelineEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode pipeline event; acking and dropping")

		return nil
	}

	finished, ok := event.Data.(*pipelineservice.WorkflowRunFinishedEventData)
	if !ok || finished == nil || finished.Status != pipelineservice.WorkflowRunStatusFailed {
		return nil
	}

	var runError string
	if finished.Error != nil {
		runError = *finished.Error
	}

	if err := h.deliverWebhook.Execute(ctx, notifyservice.DeliverWebhookParams{
		WorkspaceID: finished.WorkspaceID,
		Event: notifyservice.WebhookEvent{
			Event:         webhookEventWorkflowFailed,
			Timestamp:     time.Now(),
			WorkspaceID:   finished.WorkspaceID.String(),
			WorkflowID:    finished.WorkflowID.String(),
			WorkflowName:  finished.WorkflowName,
			WorkflowRunID: finished.RunID.String(),
			Error:         runError,
		},
	}); err != nil {
		h.logger.
			WithError(err).
			WithField("workflow_run_id", finished.RunID.String()).
			Error(ctx, "delivering webhook")
	}

	if len(finished.ConnectionIDs) == 0 {
		return nil
	}

	if err := h.deliverNotification.Execute(ctx, notifyservice.DeliverNotificationParams{
		WorkspaceID:   finished.WorkspaceID,
		ConnectionIDs: finished.ConnectionIDs,
		Event:         notifyservice.NotificationEventWorkflowFailed,
		WorkflowID:    finished.WorkflowID,
		WorkflowName:  finished.WorkflowName,
		WorkflowRunID: finished.RunID,
		Error:         runError,
	}); err != nil {
		h.logger.
			WithError(err).
			WithField("workflow_run_id", finished.RunID.String()).
			Error(ctx, "delivering channel notifications")
	}

	return nil
}
