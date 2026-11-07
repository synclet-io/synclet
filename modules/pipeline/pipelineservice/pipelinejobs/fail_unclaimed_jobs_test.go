package pipelinejobs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type unclaimedFixture struct {
	store *pipelinefake.Storage
	conn  *pipelineservice.Connection
	job   *pipelineservice.Job
	uc    *FailUnclaimedJobs
}

func newUnclaimedFixture(t *testing.T, waited time.Duration) *unclaimedFixture {
	t.Helper()

	store := pipelinefake.New()
	schedule := "0 * * * *"
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New(), Status: pipelineservice.ConnectionStatusActive, Schedule: &schedule}
	store.ConnectionsTable.Insert(conn)

	job := &pipelineservice.Job{
		ID: uuid.New(), ConnectionID: conn.ID, Status: pipelineservice.JobStatusScheduled, JobType: pipelineservice.JobTypeSync,
		ScheduledAt: time.Now().Add(-waited), Attempt: 1, MaxAttempts: 3, ExecutorGroup: "us",
	}
	store.JobsTable.Insert(job)

	cfg := pipelineservice.Config{JobClaimTimeout: 5 * time.Minute, ExecutorSilenceTimeout: time.Minute}

	return &unclaimedFixture{store: store, conn: conn, job: job, uc: NewFailUnclaimedJobs(store, cfg, nil)}
}

func (f *unclaimedFixture) eventsOf(eventType pipelineservice.PipelineEventType) []*pipelineservice.PipelineEvent {
	var out []*pipelineservice.PipelineEvent

	for _, event := range f.store.Events {
		if event.EventType == eventType {
			out = append(out, event)
		}
	}

	return out
}

func TestFailUnclaimedJobs(t *testing.T) {
	ctx := context.Background()

	t.Run("unclaimed past the timeout with no live executor fails for good", func(t *testing.T) {
		f := newUnclaimedFixture(t, 6*time.Minute)
		f.store.AddLiveExecutor("eu")

		failed, err := f.uc.Execute(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, failed)

		job := f.store.JobsTable.Get(f.job.ID)
		assert.Equal(t, pipelineservice.JobStatusFailed, job.Status)
		require.NotNil(t, job.Error)
		assert.Equal(t, `no live executor in group "us"`, *job.Error)
		assert.NotNil(t, job.CompletedAt)
		assert.Equal(t, 1, job.Attempt, "the failure is final, not a retry")
		assert.NotNil(t, f.store.ConnectionsTable.Get(f.conn.ID).NextScheduledAt, "the next run is scheduled")

		updates := f.eventsOf(pipelineservice.PipelineEventTypeJobUpdated)
		require.Len(t, updates, 1)
		data, ok := updates[0].Data.(*pipelineservice.JobUpdatedEventData)
		require.True(t, ok)
		assert.Equal(t, pipelineservice.JobStatusScheduled, data.OldData.Status)
		assert.Equal(t, pipelineservice.JobStatusFailed, data.NewData.Status)
		assert.Empty(t, f.eventsOf(pipelineservice.PipelineEventTypeJobQueueStalled))
	})

	t.Run("unclaimed past the timeout with busy executors stays queued and stalls once", func(t *testing.T) {
		f := newUnclaimedFixture(t, 6*time.Minute)
		f.store.AddLiveExecutor("us")

		failed, err := f.uc.Execute(ctx)
		require.NoError(t, err)
		assert.Zero(t, failed)

		job := f.store.JobsTable.Get(f.job.ID)
		assert.Equal(t, pipelineservice.JobStatusScheduled, job.Status)
		assert.Equal(t, 1, job.Attempt)
		require.NotNil(t, job.QueueReason)
		assert.Equal(t, `executors in group "us" are busy`, *job.QueueReason)
		require.NotNil(t, job.StalledAt)

		stalls := f.eventsOf(pipelineservice.PipelineEventTypeJobQueueStalled)
		require.Len(t, stalls, 1)
		data, ok := stalls[0].Data.(*pipelineservice.JobQueueStalledEventData)
		require.True(t, ok)
		assert.Equal(t, f.conn.WorkspaceID, data.WorkspaceID)
		assert.Equal(t, f.job.ID, data.Job.ID)
		assert.Equal(t, "us", data.Group)
		assert.GreaterOrEqual(t, data.WaitedMs, (6 * time.Minute).Milliseconds())

		_, err = f.uc.Execute(ctx)
		require.NoError(t, err)
		assert.Len(t, f.eventsOf(pipelineservice.PipelineEventTypeJobQueueStalled), 1, "no second event on the next tick")
		assert.Equal(t, pipelineservice.JobStatusScheduled, f.store.JobsTable.Get(f.job.ID).Status)
	})

	t.Run("unclaimed under the timeout is left alone", func(t *testing.T) {
		f := newUnclaimedFixture(t, time.Minute)

		failed, err := f.uc.Execute(ctx)
		require.NoError(t, err)
		assert.Zero(t, failed)
		assert.Equal(t, *f.job, *f.store.JobsTable.Get(f.job.ID))
		assert.Empty(t, f.store.Events)
	})
}
