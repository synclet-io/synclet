package pipelineevent

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

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

func ptr[T any](v T) *T {
	return &v
}

func TestEncodeDecode_JobCreatedRoundTrip(t *testing.T) {
	job := newJob(pipelineservice.JobStatusFailed)
	now := time.Now().UTC().Truncate(time.Microsecond)
	job.StartedAt = ptr(now)
	job.CompletedAt = ptr(now.Add(time.Minute))
	job.HeartbeatAt = ptr(now.Add(30 * time.Second))
	job.Error = ptr("connector exited with code 1")
	job.WorkerID = ptr("worker-1")
	job.K8sJobName = ptr("sync-abc")
	job.FailureReason = ptr("system_error")
	job.WorkflowStepRunID = ptr(uuid.New())

	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobCreated,
		Data: &pipelineservice.JobCreatedEventData{
			WorkspaceID: uuid.New(),
			Job:         job,
		},
	}

	metadata, body, err := EncodePipelineEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "job.created", metadata[eventutil.MetadataKeyEventType])
	assert.NotEmpty(t, body)

	decoded, err := DecodePipelineEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded), "round-trip must preserve event equality")
}

func TestEncodeDecode_JobUpdatedRoundTrip(t *testing.T) {
	oldJob := newJob(pipelineservice.JobStatusScheduled)
	updatedJob := oldJob.Copy()
	updatedJob.Status = pipelineservice.JobStatusRunning
	updatedJob.WorkerID = ptr("worker-1")
	updatedJob.StartedAt = ptr(time.Now().UTC().Truncate(time.Microsecond))

	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobUpdated,
		Data: &pipelineservice.JobUpdatedEventData{
			WorkspaceID: uuid.New(),
			OldData:     oldJob,
			NewData:     &updatedJob,
		},
	}

	metadata, body, err := EncodePipelineEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "job.updated", metadata[eventutil.MetadataKeyEventType])

	decoded, err := DecodePipelineEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded))
}

func TestEncodeDecode_WorkflowRunStartedRoundTrip(t *testing.T) {
	data := &pipelineservice.WorkflowRunStartedEventData{WorkspaceID: uuid.New(), WorkflowID: uuid.New(), RunID: uuid.New()}
	event := &pipelineservice.PipelineEvent{EventType: pipelineservice.PipelineEventTypeWorkflowRunStarted, Data: data}

	metadata, body, err := EncodePipelineEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "workflow_run.started", metadata[eventutil.MetadataKeyEventType])

	decoded, err := DecodePipelineEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded))

	key, err := OrderingKey(event)
	require.NoError(t, err)
	assert.Equal(t, data.WorkflowID.String(), key, "runs of one workflow stay in order")
}

func TestOrderingKey_JobCreated(t *testing.T) {
	job := newJob(pipelineservice.JobStatusScheduled)
	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobCreated,
		Data:      &pipelineservice.JobCreatedEventData{WorkspaceID: uuid.New(), Job: job},
	}

	got, err := OrderingKey(event)
	require.NoError(t, err)
	assert.Equal(t, job.ConnectionID.String(), got)
}

func TestOrderingKey_JobUpdatedUsesNewConnectionID(t *testing.T) {
	oldJob := newJob(pipelineservice.JobStatusScheduled)
	updatedJob := newJob(pipelineservice.JobStatusRunning)
	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobUpdated,
		Data:      &pipelineservice.JobUpdatedEventData{WorkspaceID: uuid.New(), OldData: oldJob, NewData: updatedJob},
	}

	got, err := OrderingKey(event)
	require.NoError(t, err)
	assert.Equal(t, updatedJob.ConnectionID.String(), got)
}

func TestOrderingKey_NilEvent(t *testing.T) {
	_, err := OrderingKey(nil)
	require.Error(t, err)
}

func TestEncode_NilEvent(t *testing.T) {
	_, _, err := EncodePipelineEvent(nil)
	require.Error(t, err)
}

func TestEncode_JobCreatedMissingJob(t *testing.T) {
	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobCreated,
		Data:      &pipelineservice.JobCreatedEventData{WorkspaceID: uuid.New()},
	}

	_, _, err := EncodePipelineEvent(event)
	require.Error(t, err)
}

func TestEncode_JobUpdatedMissingNewData(t *testing.T) {
	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobUpdated,
		Data: &pipelineservice.JobUpdatedEventData{
			WorkspaceID: uuid.New(),
			OldData:     newJob(pipelineservice.JobStatusScheduled),
		},
	}

	_, _, err := EncodePipelineEvent(event)
	require.Error(t, err)
}

func TestEncode_UnsupportedEventType(t *testing.T) {
	event := &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventType(99),
		Data: &pipelineservice.JobCreatedEventData{
			WorkspaceID: uuid.New(),
			Job:         newJob(pipelineservice.JobStatusScheduled),
		},
	}

	_, _, err := EncodePipelineEvent(event)
	require.Error(t, err)
}

func TestDecode_MissingMetadataKey(t *testing.T) {
	_, err := DecodePipelineEvent(map[string]string{}, []byte("{}"))
	require.Error(t, err)
}

func TestDecode_UnknownEventType(t *testing.T) {
	_, err := DecodePipelineEvent(map[string]string{
		eventutil.MetadataKeyEventType: "job.unknown",
	}, []byte("{}"))
	require.Error(t, err)
}

func TestDecode_BadJSON(t *testing.T) {
	_, err := DecodePipelineEvent(map[string]string{
		eventutil.MetadataKeyEventType: "job.created",
	}, []byte("not-json"))
	require.Error(t, err)
}

func TestJobTypeFromString(t *testing.T) {
	for _, jobType := range []pipelineservice.JobType{
		pipelineservice.JobTypeSync,
		pipelineservice.JobTypeDiscover,
		pipelineservice.JobTypeCheck,
	} {
		got, err := jobTypeFromString(jobType.String())
		require.NoError(t, err)
		assert.Equal(t, jobType, got)
	}

	_, err := jobTypeFromString("unknown")
	require.Error(t, err)
}

func TestJobStatusFromString(t *testing.T) {
	for _, status := range []pipelineservice.JobStatus{
		pipelineservice.JobStatusScheduled,
		pipelineservice.JobStatusStarting,
		pipelineservice.JobStatusRunning,
		pipelineservice.JobStatusCompleted,
		pipelineservice.JobStatusFailed,
		pipelineservice.JobStatusCancelled,
	} {
		got, err := jobStatusFromString(status.String())
		require.NoError(t, err)
		assert.Equal(t, status, got)
	}

	_, err := jobStatusFromString("unknown")
	require.Error(t, err)
}

func TestEncodeDecode_WorkflowRunFinishedRoundTrip(t *testing.T) {
	reason := "boom"
	data := &pipelineservice.WorkflowRunFinishedEventData{
		WorkspaceID:   uuid.New(),
		WorkflowID:    uuid.New(),
		WorkflowName:  "nightly",
		RunID:         uuid.New(),
		Status:        pipelineservice.WorkflowRunStatusFailed,
		Error:         &reason,
		ConnectionIDs: []uuid.UUID{uuid.New(), uuid.New()},
	}
	event := &pipelineservice.PipelineEvent{EventType: pipelineservice.PipelineEventTypeWorkflowRunFinished, Data: data}

	metadata, body, err := EncodePipelineEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "workflow_run.finished", metadata[eventutil.MetadataKeyEventType])

	decoded, err := DecodePipelineEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded))

	key, err := OrderingKey(event)
	require.NoError(t, err)
	assert.Equal(t, data.WorkflowID.String(), key)
}
