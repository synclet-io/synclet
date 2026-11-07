package notifyeventhandle

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestJobStalledEventHandler(t *testing.T) {
	workspaceID, connID := uuid.New(), uuid.New()
	stalledChannel, failureChannel := uuid.New(), uuid.New()
	subscribedWebhook, otherWebhook := "https://8.8.8.8/stalled", "https://8.8.8.8/failed"

	channel := func(id uuid.UUID) *notifyservice.NotificationChannel {
		return &notifyservice.NotificationChannel{ID: id, WorkspaceID: workspaceID, ChannelType: notifyservice.ChannelTypeSlack, Enabled: true}
	}
	rule := func(channelID uuid.UUID, condition notifyservice.NotificationCondition) *notifyservice.NotificationRule {
		return &notifyservice.NotificationRule{ID: uuid.New(), WorkspaceID: workspaceID, ChannelID: channelID, ConnectionID: connID, Condition: condition, Enabled: true}
	}

	store := &notifyStore{
		webhooks: []*notifyservice.Webhook{
			{ID: uuid.New(), WorkspaceID: workspaceID, URL: subscribedWebhook, Events: `["sync.stalled"]`, Enabled: true},
			{ID: uuid.New(), WorkspaceID: workspaceID, URL: otherWebhook, Events: `["sync.failed"]`, Enabled: true},
		},
		channels: []*notifyservice.NotificationChannel{channel(stalledChannel), channel(failureChannel)},
		rules: []*notifyservice.NotificationRule{
			rule(stalledChannel, notifyservice.NotificationConditionOnJobStalled),
			rule(failureChannel, notifyservice.NotificationConditionOnFailure),
		},
	}

	transport := &recordingTransport{requests: map[string][]byte{}}
	channels := &recordingChannel{deliveries: map[uuid.UUID]notifyservice.WebhookEvent{}}
	logger := &logging.Logger{Delegate: noopLogDelegate{}}

	handler := NewJobStalledEventHandler(
		logger,
		notifyservice.NewDeliverWebhookWithClient(store, nil, notifyservice.Config{WebhookMaxRetries: 1}, &http.Client{Transport: transport}),
		notifyservice.NewDeliverNotification(store, map[notifyservice.ChannelType]notifyservice.ChannelDeliverer{
			notifyservice.ChannelTypeSlack: channels,
		}, logger),
	)

	job := &pipelineservice.Job{
		ID: uuid.New(), ConnectionID: connID, Status: pipelineservice.JobStatusScheduled, JobType: pipelineservice.JobTypeSync,
		ScheduledAt: time.Now().UTC().Truncate(time.Microsecond), Attempt: 1, MaxAttempts: 3, ExecutorGroup: "us",
	}

	require.NoError(t, handler.Handle(encode(t, &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobQueueStalled,
		Data: &pipelineservice.JobQueueStalledEventData{
			WorkspaceID: workspaceID, Job: job, Group: "us", WaitedMs: 300000,
		},
	})))

	require.Contains(t, transport.requests, subscribedWebhook)
	assert.NotContains(t, transport.requests, otherWebhook)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(transport.requests[subscribedWebhook], &payload))
	assert.Equal(t, "sync.stalled", payload["event"])
	assert.Equal(t, workspaceID.String(), payload["workspace_id"])
	assert.Equal(t, connID.String(), payload["connection_id"])
	assert.Equal(t, job.ID.String(), payload["job_id"])
	assert.Equal(t, "us", payload["executor_group"])
	assert.InDelta(t, 300000, payload["waited_ms"], 0)

	require.Contains(t, channels.deliveries, stalledChannel, "an OnJobStalled rule delivers")
	assert.NotContains(t, channels.deliveries, failureChannel, "an OnFailure rule does not match a stalled job")
	assert.Equal(t, "sync.stalled", channels.deliveries[stalledChannel].Event)
	assert.Equal(t, "us", channels.deliveries[stalledChannel].ExecutorGroup)
}
