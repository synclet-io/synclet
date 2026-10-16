package pipelineeventhandle

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type stubArchiver struct {
	called []uuid.UUID
	err    error
}

func (s *stubArchiver) Execute(_ context.Context, jobID uuid.UUID) error {
	s.called = append(s.called, jobID)

	return s.err
}

func newArchiveHandler(stub *stubArchiver) *JobLogArchiveHandler {
	return &JobLogArchiveHandler{logger: (*logging.Logger)(nil), archive: stub}
}

func jobUpdatedMessage(t *testing.T, jobID uuid.UUID, from, to pipelineservice.JobStatus) *message.Message {
	t.Helper()

	metadata, body, err := pipelineevent.EncodePipelineEvent(&pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobUpdated,
		Data: &pipelineservice.JobUpdatedEventData{
			WorkspaceID: uuid.New(),
			OldData:     &pipelineservice.Job{ID: jobID, JobType: pipelineservice.JobTypeSync, Status: from},
			NewData:     &pipelineservice.Job{ID: jobID, JobType: pipelineservice.JobTypeSync, Status: to},
		},
	})
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	return msg
}

func TestJobLogArchiveHandler_ArchivesFinishedJobs(t *testing.T) {
	for _, status := range []pipelineservice.JobStatus{
		pipelineservice.JobStatusCompleted,
		pipelineservice.JobStatusFailed,
		pipelineservice.JobStatusCancelled,
	} {
		t.Run(status.String(), func(t *testing.T) {
			jobID := uuid.New()
			stub := &stubArchiver{}

			require.NoError(t, newArchiveHandler(stub).Handle(jobUpdatedMessage(t, jobID, pipelineservice.JobStatusRunning, status)))
			assert.Equal(t, []uuid.UUID{jobID}, stub.called)
		})
	}
}

func TestJobLogArchiveHandler_IgnoresUnfinishedAndUnchanged(t *testing.T) {
	stub := &stubArchiver{}
	handler := newArchiveHandler(stub)

	require.NoError(t, handler.Handle(jobUpdatedMessage(t, uuid.New(), pipelineservice.JobStatusRunning, pipelineservice.JobStatusScheduled)))
	require.NoError(t, handler.Handle(jobUpdatedMessage(t, uuid.New(), pipelineservice.JobStatusCompleted, pipelineservice.JobStatusCompleted)))
	require.NoError(t, handler.Handle(message.NewMessage(uuid.NewString(), []byte("{}"))))

	assert.Empty(t, stub.called)
}

func TestJobLogArchiveHandler_ArchiveErrorIsRetried(t *testing.T) {
	stub := &stubArchiver{err: errors.New("bucket down")}

	err := newArchiveHandler(stub).Handle(jobUpdatedMessage(t, uuid.New(), pipelineservice.JobStatusRunning, pipelineservice.JobStatusFailed))
	require.ErrorContains(t, err, "bucket down")
}

func TestJobLogArchiveHandler_Metadata(t *testing.T) {
	handler := newArchiveHandler(&stubArchiver{})

	assert.Equal(t, "pipeline_job_log_archive", handler.Name())
	assert.Equal(t, pipelineevent.PipelineEventTopic, handler.Topic())
}
