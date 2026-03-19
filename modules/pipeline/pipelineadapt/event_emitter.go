package pipelineadapt

import (
	"context"
	"time"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// EventEmitterAdapter adapts notifyservice.DeliverWebhook use case to pipelineservice.SyncEventEmitter.
type EventEmitterAdapter struct {
	deliverWebhook *notifyservice.DeliverWebhook
}

// NewEventEmitterAdapter creates a new EventEmitterAdapter.
func NewEventEmitterAdapter(deliverWebhook *notifyservice.DeliverWebhook) *EventEmitterAdapter {
	return &EventEmitterAdapter{deliverWebhook: deliverWebhook}
}

func (a *EventEmitterAdapter) EmitSyncCompleted(ctx context.Context, event pipelineservice.SyncCompletedEvent) error {
	return a.deliverWebhook.Execute(ctx, notifyservice.DeliverWebhookParams{
		WorkspaceID: event.WorkspaceID,
		Event: notifyservice.WebhookEvent{
			Event:        "sync.completed",
			Timestamp:    time.Now(),
			ConnectionID: event.ConnectionID.String(),
			JobID:        event.JobID.String(),
		},
	})
}

func (a *EventEmitterAdapter) EmitSyncFailed(ctx context.Context, event pipelineservice.SyncFailedEvent) error {
	return a.deliverWebhook.Execute(ctx, notifyservice.DeliverWebhookParams{
		WorkspaceID: event.WorkspaceID,
		Event: notifyservice.WebhookEvent{
			Event:        "sync.failed",
			Timestamp:    time.Now(),
			ConnectionID: event.ConnectionID.String(),
			JobID:        event.JobID.String(),
			Error:        event.Error,
		},
	})
}
