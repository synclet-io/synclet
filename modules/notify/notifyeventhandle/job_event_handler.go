// Package notifyeventhandle contains Watermill handlers that react to
// inter-module events. Handlers here treat the producing module's
// {module}event package as a public contract; they never reach into the
// producer's storage or service packages directly.
package notifyeventhandle

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// JobEventHandler reacts to pipeline job events. On terminal transitions
// (running/starting → completed | failed | cancelled) it fans out
// webhook deliveries and channel notifications. Intermediate transitions
// (scheduled → starting, heartbeat updates, retries) are intentionally
// ignored so notification consumers only see end-of-sync signals.
type JobEventHandler struct {
	logger              *logging.Logger
	deliverWebhook      webhookDeliverer
	deliverNotification notificationDeliverer
	jobInfoLoader       notifyservice.JobInfoLoader
}

type webhookDeliverer interface {
	Execute(ctx context.Context, params notifyservice.DeliverWebhookParams) error
}

type notificationDeliverer interface {
	Execute(ctx context.Context, params notifyservice.DeliverNotificationParams) error
}

var _ pnpwatermill.Handler = (*JobEventHandler)(nil)

// NewJobEventHandler creates the handler. Wired via pnpwatermill.HandlerProvider
// in the notify FX module.
func NewJobEventHandler(
	logger *logging.Logger,
	deliverWebhook *notifyservice.DeliverWebhook,
	deliverNotification *notifyservice.DeliverNotification,
	jobInfoLoader notifyservice.JobInfoLoader,
) *JobEventHandler {
	return &JobEventHandler{
		logger:              logger.Named("job-event-handler"),
		deliverWebhook:      deliverWebhook,
		deliverNotification: deliverNotification,
		jobInfoLoader:       jobInfoLoader,
	}
}

func (h *JobEventHandler) Name() string {
	return "notify_pipeline_job_event"
}

func (h *JobEventHandler) Topic() string {
	return pipelineevent.PipelineEventTopic
}

// Handle decodes the message and, for JobUpdated terminal transitions, fans
// out webhook + channel deliveries. Decode failures and delivery failures are
// logged and the message is acked.
func (h *JobEventHandler) Handle(msg *message.Message) error {
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

	// Only react to status transitions, not heartbeat-only or worker-id changes.
	if updated.OldData.Status == updated.NewData.Status {
		return nil
	}

	eventName := webhookEventNameFor(updated.NewData.Status)
	if eventName == "" {
		return nil
	}

	job := updated.NewData

	info, err := h.jobInfoLoader.LoadJobInfo(ctx, updated.WorkspaceID, job.ConnectionID, job.ID)
	if err != nil {
		h.logger.
			WithError(err).
			WithField("job_id", job.ID.String()).
			Warn(ctx, "loading job info for event enrichment; continuing with partial payload")
	}

	payload := notifyservice.WebhookEvent{
		Event:        eventName,
		Timestamp:    time.Now(),
		WorkspaceID:  updated.WorkspaceID.String(),
		ConnectionID: job.ConnectionID.String(),
		JobID:        job.ID.String(),
		JobType:      job.JobType.String(),
		Attempt:      job.Attempt,
	}

	if job.Error != nil {
		payload.Error = *job.Error
	}

	if info != nil {
		payload.ConnectionName = info.ConnectionName
		payload.SourceName = info.SourceName
		payload.DestinationName = info.DestinationName

		if info.LatestAttempt != nil {
			payload.Attempt = info.LatestAttempt.AttemptNumber
			payload.RecordsRead = info.LatestAttempt.RecordsRead
			payload.BytesSynced = info.LatestAttempt.BytesSynced
			payload.DurationMs = info.LatestAttempt.DurationMs
		}
	}

	if err := h.deliverWebhook.Execute(ctx, notifyservice.DeliverWebhookParams{
		WorkspaceID: updated.WorkspaceID,
		Event:       payload,
	}); err != nil {
		h.logger.
			WithError(err).
			WithField("job_id", job.ID.String()).
			Error(ctx, "delivering webhook")
	}

	notificationEvent, ok := notificationEventFor(updated.NewData.Status)
	if ok {
		var recordsSynced int64
		if info != nil && info.LatestAttempt != nil {
			recordsSynced = info.LatestAttempt.RecordsRead
		}

		if err := h.deliverNotification.Execute(ctx, notifyservice.DeliverNotificationParams{
			WorkspaceID:   updated.WorkspaceID,
			ConnectionID:  job.ConnectionID,
			Event:         notificationEvent,
			RecordsSynced: recordsSynced,
		}); err != nil {
			h.logger.
				WithError(err).
				WithField("job_id", job.ID.String()).
				Error(ctx, "delivering channel notifications")
		}
	}

	return nil
}

func webhookEventNameFor(status pipelineservice.JobStatus) string {
	switch status {
	case pipelineservice.JobStatusCompleted:
		return "sync.completed"
	case pipelineservice.JobStatusFailed:
		return "sync.failed"
	default:
		return ""
	}
}

func notificationEventFor(status pipelineservice.JobStatus) (notifyservice.NotificationEvent, bool) {
	switch status {
	case pipelineservice.JobStatusCompleted:
		return notifyservice.NotificationEventCompleted, true
	case pipelineservice.JobStatusFailed:
		return notifyservice.NotificationEventFailed, true
	default:
		return 0, false
	}
}
