package bus_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/notify/notifyeventhandle"
	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/logarchive"
)

// TestReportCompletion_NotBlockedBySlowWebhook shows that reporting a sync
// completion only writes the job update and its outbox event, while the
// webhook delivery happens later in the event consumer: the completion call
// returns while the webhook endpoint is still hanging and has not been called.
func TestReportCompletion_NotBlockedBySlowWebhook(t *testing.T) {
	transport := newBlockingTransport()

	workspaceID := uuid.New()
	startedAt := time.Now().Add(-time.Minute)
	job := &pipelineservice.Job{
		ID:           uuid.New(),
		ConnectionID: uuid.New(),
		Status:       pipelineservice.JobStatusRunning,
		JobType:      pipelineservice.JobTypeSync,
		ScheduledAt:  startedAt,
		StartedAt:    &startedAt,
		Attempt:      1,
		MaxAttempts:  3,
		CreatedAt:    startedAt,
	}
	conn := &pipelineservice.Connection{
		ID:          job.ConnectionID,
		WorkspaceID: workspaceID,
		Name:        "pg-to-bq",
		Status:      pipelineservice.ConnectionStatusActive,
		MaxAttempts: 3,
	}

	store := newPipelineStore(job, conn)
	logger := testLogger()
	cfg := pipelineservice.Config{EventEmitTimeout: time.Second}
	reportCompletion := pipelinejobs.NewReportCompletion(
		pipelinejobs.NewUpdateJobStatus(store),
		store,
		pipelinejobs.NewCleanupOldJobs(store, unlimitedRetention{}, unlimitedRetention{}, logarchive.None{}, logger),
		logger,
		cfg,
	)

	t.Cleanup(reportCompletion.Wait)

	handler := notifyeventhandle.NewJobEventHandler(
		logger,
		notifyservice.NewDeliverWebhookWithClient(newNotifyStore(&notifyservice.Webhook{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			URL:         "http://203.0.113.10/hooks/sync",
			Events:      `["sync.completed"]`,
			Enabled:     true,
		}), nil, notifyservice.Config{WebhookHTTPTimeout: 5 * time.Second, WebhookMaxRetries: 1}, &http.Client{Transport: transport}),
		notifyservice.NewDeliverNotification(newNotifyStore(nil), nil, logger),
		noJobInfo{},
	)

	ctx := context.Background()
	started := time.Now()
	err := reportCompletion.Execute(ctx, pipelinejobs.ReportCompletionParams{
		JobID:        job.ID,
		ConnectionID: job.ConnectionID,
		Success:      true,
		RecordsRead:  1500,
		BytesSynced:  4096,
		DurationMs:   3200,
	})
	elapsed := time.Since(started)

	require.NoError(t, err)
	assert.Less(t, elapsed, 100*time.Millisecond)
	assert.Equal(t, int32(0), transport.calls.Load())

	events := store.outbox.recorded()
	require.Len(t, events, 1)

	metadata, body, err := pipelineevent.EncodePipelineEvent(events[0])
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	msg.Metadata = metadata

	handled := make(chan error, 1)

	go func() {
		handled <- handler.Handle(msg)
	}()

	select {
	case <-transport.entered:
	case <-time.After(time.Second):
		t.Fatal("webhook delivery never reached the HTTP transport")
	}

	select {
	case err := <-handled:
		t.Fatalf("handler returned %v while the webhook endpoint was still blocked", err)
	default:
	}

	close(transport.release)

	select {
	case err := <-handled:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("handler did not finish after the webhook endpoint was released")
	}

	bodies := transport.requestBodies()
	require.Len(t, bodies, 1)
	assert.Equal(t, int32(1), transport.calls.Load())

	var delivered notifyservice.WebhookEvent

	require.NoError(t, json.Unmarshal(bodies[0], &delivered))
	assert.Equal(t, "sync.completed", delivered.Event)
	assert.Equal(t, job.ID.String(), delivered.JobID)
	assert.Equal(t, workspaceID.String(), delivered.WorkspaceID)
}
