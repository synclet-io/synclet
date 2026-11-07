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

const webhookEventSyncStalled = "sync.stalled"

// JobStalledEventHandler reacts to job.queue_stalled: it sends the
// sync.stalled webhook event with the executor group and the time the job has
// waited, and notifies the channels of OnJobStalled rules covering the
// connection or the whole workspace.
type JobStalledEventHandler struct {
	logger              *logging.Logger
	deliverWebhook      webhookDeliverer
	deliverNotification notificationDeliverer
}

var _ pnpwatermill.Handler = (*JobStalledEventHandler)(nil)

// NewJobStalledEventHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the notify FX module.
func NewJobStalledEventHandler(
	logger *logging.Logger,
	deliverWebhook *notifyservice.DeliverWebhook,
	deliverNotification *notifyservice.DeliverNotification,
) *JobStalledEventHandler {
	return &JobStalledEventHandler{
		logger:              logger.Named("job-stalled-event-handler"),
		deliverWebhook:      deliverWebhook,
		deliverNotification: deliverNotification,
	}
}

func (h *JobStalledEventHandler) Name() string {
	return "notify_pipeline_job_stalled_event"
}

func (h *JobStalledEventHandler) Topic() string {
	return pipelineevent.PipelineEventTopic
}

// Handle delivers a stalled job. Decode and delivery failures are logged and
// the message is acked.
func (h *JobStalledEventHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := pipelineevent.DecodePipelineEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode pipeline event; acking and dropping")

		return nil
	}

	stalled, ok := event.Data.(*pipelineservice.JobQueueStalledEventData)
	if !ok || stalled == nil || stalled.Job == nil {
		return nil
	}

	job := stalled.Job

	if err := h.deliverWebhook.Execute(ctx, notifyservice.DeliverWebhookParams{
		WorkspaceID: stalled.WorkspaceID,
		Event: notifyservice.WebhookEvent{
			Event:         webhookEventSyncStalled,
			Timestamp:     time.Now(),
			WorkspaceID:   stalled.WorkspaceID.String(),
			ConnectionID:  job.ConnectionID.String(),
			JobID:         job.ID.String(),
			JobType:       job.JobType.String(),
			Attempt:       job.Attempt,
			ExecutorGroup: stalled.Group,
			WaitedMs:      stalled.WaitedMs,
		},
	}); err != nil {
		h.logger.WithError(err).WithField("job_id", job.ID.String()).Error(ctx, "delivering webhook")
	}

	if err := h.deliverNotification.Execute(ctx, notifyservice.DeliverNotificationParams{
		WorkspaceID:   stalled.WorkspaceID,
		ConnectionID:  job.ConnectionID,
		Event:         notifyservice.NotificationEventJobStalled,
		JobID:         job.ID,
		ExecutorGroup: stalled.Group,
		WaitedMs:      stalled.WaitedMs,
	}); err != nil {
		h.logger.WithError(err).WithField("job_id", job.ID.String()).Error(ctx, "delivering channel notifications")
	}

	return nil
}
