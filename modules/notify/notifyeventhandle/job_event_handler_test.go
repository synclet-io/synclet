package notifyeventhandle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

var errBoom = errors.New("boom")

type fakeWebhooks struct {
	calls []notifyservice.DeliverWebhookParams
	err   error
}

func (f *fakeWebhooks) Execute(_ context.Context, params notifyservice.DeliverWebhookParams) error {
	f.calls = append(f.calls, params)

	return f.err
}

type fakeNotifications struct {
	calls []notifyservice.DeliverNotificationParams
}

func (f *fakeNotifications) Execute(_ context.Context, params notifyservice.DeliverNotificationParams) error {
	f.calls = append(f.calls, params)

	return nil
}

type fakeJobInfoLoader struct {
	info  *notifyservice.JobInfo
	err   error
	calls int
}

func (f *fakeJobInfoLoader) LoadJobInfo(_ context.Context, _, _, _ uuid.UUID) (*notifyservice.JobInfo, error) {
	f.calls++

	return f.info, f.err
}

type noopLogDelegate struct{}

func (noopLogDelegate) Info(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopLogDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopLogDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopLogDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopLogDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopLogDelegate) Named(string) logging.Delegate                      { return d }
func (d noopLogDelegate) SkipCallers(int) logging.Delegate                   { return d }

type handlerFixture struct {
	handler       *JobEventHandler
	webhooks      *fakeWebhooks
	notifications *fakeNotifications
	loader        *fakeJobInfoLoader
}

func newFixture() *handlerFixture {
	f := &handlerFixture{
		webhooks:      &fakeWebhooks{},
		notifications: &fakeNotifications{},
		loader: &fakeJobInfoLoader{info: &notifyservice.JobInfo{
			ConnectionName:  "pg-to-bq",
			SourceName:      "postgres",
			DestinationName: "bigquery",
			LatestAttempt: &notifyservice.JobAttemptInfo{
				AttemptNumber: 2,
				RecordsRead:   1500,
				BytesSynced:   4096,
				DurationMs:    3200,
			},
		}},
	}

	f.handler = &JobEventHandler{
		logger:              &logging.Logger{Delegate: noopLogDelegate{}},
		deliverWebhook:      f.webhooks,
		deliverNotification: f.notifications,
		jobInfoLoader:       f.loader,
	}

	return f
}

func newJob(status pipelineservice.JobStatus) *pipelineservice.Job {
	now := time.Now().UTC().Truncate(time.Microsecond)

	return &pipelineservice.Job{
		ID:           uuid.New(),
		ConnectionID: uuid.New(),
		Status:       status,
		JobType:      pipelineservice.JobTypeSync,
		ScheduledAt:  now,
		Attempt:      1,
		MaxAttempts:  3,
		CreatedAt:    now,
	}
}

func transition(job *pipelineservice.Job, to pipelineservice.JobStatus) *pipelineservice.Job {
	next := job.Copy()
	next.Status = to

	return &next
}

func encode(t *testing.T, event *pipelineservice.PipelineEvent) *message.Message {
	t.Helper()

	metadata, body, err := pipelineevent.EncodePipelineEvent(event)
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	msg.Metadata = metadata

	return msg
}

func updatedMessage(t *testing.T, ws uuid.UUID, oldJob, newJob *pipelineservice.Job) *message.Message {
	t.Helper()

	return encode(t, &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobUpdated,
		Data: &pipelineservice.JobUpdatedEventData{
			WorkspaceID: ws,
			OldData:     oldJob,
			NewData:     newJob,
		},
	})
}

func TestJobEventHandler_Completed(t *testing.T) {
	f := newFixture()
	ws := uuid.New()
	oldJob := newJob(pipelineservice.JobStatusStarting)

	require.NoError(t, f.handler.Handle(updatedMessage(t, ws, oldJob, transition(oldJob, pipelineservice.JobStatusCompleted))))

	require.Len(t, f.webhooks.calls, 1)
	call := f.webhooks.calls[0]
	assert.Equal(t, ws, call.WorkspaceID)

	event := call.Event
	assert.False(t, event.Timestamp.IsZero())
	event.Timestamp = time.Time{}
	assert.Equal(t, notifyservice.WebhookEvent{
		Event:           "sync.completed",
		WorkspaceID:     ws.String(),
		ConnectionID:    oldJob.ConnectionID.String(),
		ConnectionName:  "pg-to-bq",
		SourceName:      "postgres",
		DestinationName: "bigquery",
		JobID:           oldJob.ID.String(),
		JobType:         pipelineservice.JobTypeSync.String(),
		Attempt:         2,
		RecordsRead:     1500,
		BytesSynced:     4096,
		DurationMs:      3200,
	}, event)

	require.Len(t, f.notifications.calls, 1)
	assert.Equal(t, notifyservice.DeliverNotificationParams{
		WorkspaceID:   ws,
		ConnectionID:  oldJob.ConnectionID,
		Event:         notifyservice.NotificationEventCompleted,
		RecordsSynced: 1500,
	}, f.notifications.calls[0])
}

func TestJobEventHandler_Failed(t *testing.T) {
	f := newFixture()
	ws := uuid.New()
	oldJob := newJob(pipelineservice.JobStatusRunning)
	failed := transition(oldJob, pipelineservice.JobStatusFailed)
	errMsg := "source unreachable"
	failed.Error = &errMsg

	require.NoError(t, f.handler.Handle(updatedMessage(t, ws, oldJob, failed)))

	require.Len(t, f.webhooks.calls, 1)
	assert.Equal(t, "sync.failed", f.webhooks.calls[0].Event.Event)
	assert.Equal(t, "source unreachable", f.webhooks.calls[0].Event.Error)

	require.Len(t, f.notifications.calls, 1)
	assert.Equal(t, notifyservice.NotificationEventFailed, f.notifications.calls[0].Event)
	assert.Equal(t, oldJob.ConnectionID, f.notifications.calls[0].ConnectionID)
}

func TestJobEventHandler_IgnoredEvents(t *testing.T) {
	ws := uuid.New()

	cases := map[string]func(t *testing.T) *message.Message{
		"running to cancelled": func(t *testing.T) *message.Message {
			t.Helper()

			job := newJob(pipelineservice.JobStatusRunning)

			return updatedMessage(t, ws, job, transition(job, pipelineservice.JobStatusCancelled))
		},
		"scheduled to starting": func(t *testing.T) *message.Message {
			t.Helper()

			job := newJob(pipelineservice.JobStatusScheduled)

			return updatedMessage(t, ws, job, transition(job, pipelineservice.JobStatusStarting))
		},
		"heartbeat-only update": func(t *testing.T) *message.Message {
			t.Helper()

			job := newJob(pipelineservice.JobStatusRunning)
			next := transition(job, pipelineservice.JobStatusRunning)
			beat := time.Now().UTC().Truncate(time.Microsecond)
			next.HeartbeatAt = &beat

			return updatedMessage(t, ws, job, next)
		},
		"job created": func(t *testing.T) *message.Message {
			t.Helper()

			return encode(t, &pipelineservice.PipelineEvent{
				EventType: pipelineservice.PipelineEventTypeJobCreated,
				Data: &pipelineservice.JobCreatedEventData{
					WorkspaceID: ws,
					Job:         newJob(pipelineservice.JobStatusScheduled),
				},
			})
		},
		"undecodable payload": func(*testing.T) *message.Message {
			msg := message.NewMessage(uuid.NewString(), []byte("{not json"))
			msg.Metadata = message.Metadata{eventutil.MetadataKeyEventType: "job.updated"}

			return msg
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture()

			require.NoError(t, f.handler.Handle(build(t)))
			assert.Empty(t, f.webhooks.calls)
			assert.Empty(t, f.notifications.calls)
			assert.Zero(t, f.loader.calls)
		})
	}
}

func TestJobEventHandler_JobInfoLoaderErrorDeliversPartialPayload(t *testing.T) {
	f := newFixture()
	f.loader.info = nil
	f.loader.err = errBoom
	ws := uuid.New()
	oldJob := newJob(pipelineservice.JobStatusRunning)

	require.NoError(t, f.handler.Handle(updatedMessage(t, ws, oldJob, transition(oldJob, pipelineservice.JobStatusCompleted))))

	require.Len(t, f.webhooks.calls, 1)
	event := f.webhooks.calls[0].Event
	assert.Equal(t, "sync.completed", event.Event)
	assert.Equal(t, oldJob.ID.String(), event.JobID)
	assert.Equal(t, 1, event.Attempt)
	assert.Empty(t, event.ConnectionName)
	assert.Zero(t, event.RecordsRead)

	require.Len(t, f.notifications.calls, 1)
	assert.Zero(t, f.notifications.calls[0].RecordsSynced)
}

func TestJobEventHandler_WebhookErrorStillNotifies(t *testing.T) {
	f := newFixture()
	f.webhooks.err = errBoom
	ws := uuid.New()
	oldJob := newJob(pipelineservice.JobStatusRunning)

	require.NoError(t, f.handler.Handle(updatedMessage(t, ws, oldJob, transition(oldJob, pipelineservice.JobStatusFailed))))

	assert.Len(t, f.webhooks.calls, 1)
	require.Len(t, f.notifications.calls, 1)
	assert.Equal(t, notifyservice.NotificationEventFailed, f.notifications.calls[0].Event)
}
