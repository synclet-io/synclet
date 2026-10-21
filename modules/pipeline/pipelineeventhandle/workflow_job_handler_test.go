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

type stubWorkflowAdvancer struct {
	jobs []*pipelineservice.Job
	err  error
}

func (s *stubWorkflowAdvancer) ExecuteForJob(_ context.Context, job *pipelineservice.Job) error {
	s.jobs = append(s.jobs, job)

	return s.err
}

func workflowJobMessage(t *testing.T, stepRunID *uuid.UUID, from, to pipelineservice.JobStatus) *message.Message {
	t.Helper()

	jobID, connID := uuid.New(), uuid.New()
	metadata, body, err := pipelineevent.EncodePipelineEvent(&pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobUpdated,
		Data: &pipelineservice.JobUpdatedEventData{
			WorkspaceID: uuid.New(),
			OldData:     &pipelineservice.Job{ID: jobID, ConnectionID: connID, JobType: pipelineservice.JobTypeSync, Status: from, WorkflowStepRunID: stepRunID},
			NewData:     &pipelineservice.Job{ID: jobID, ConnectionID: connID, JobType: pipelineservice.JobTypeSync, Status: to, WorkflowStepRunID: stepRunID},
		},
	})
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	return msg
}

func TestWorkflowJobHandler_AdvancesOnFinalStatus(t *testing.T) {
	for _, status := range []pipelineservice.JobStatus{
		pipelineservice.JobStatusCompleted,
		pipelineservice.JobStatusFailed,
		pipelineservice.JobStatusCancelled,
	} {
		t.Run(status.String(), func(t *testing.T) {
			stub := &stubWorkflowAdvancer{}
			handler := &WorkflowJobHandler{logger: (*logging.Logger)(nil), advance: stub}
			stepRunID := uuid.New()

			require.NoError(t, handler.Handle(workflowJobMessage(t, &stepRunID, pipelineservice.JobStatusRunning, status)))
			require.Len(t, stub.jobs, 1)
			require.NotNil(t, stub.jobs[0].WorkflowStepRunID, "the event carries the job's step run")
			assert.Equal(t, stepRunID, *stub.jobs[0].WorkflowStepRunID)
		})
	}
}

func TestWorkflowJobHandler_IgnoresNonFinalStatus(t *testing.T) {
	stub := &stubWorkflowAdvancer{}
	handler := &WorkflowJobHandler{logger: (*logging.Logger)(nil), advance: stub}

	require.NoError(t, handler.Handle(workflowJobMessage(t, nil, pipelineservice.JobStatusRunning, pipelineservice.JobStatusScheduled)))
	assert.Empty(t, stub.jobs, "a job between attempts does not advance runs")
}

func TestWorkflowJobHandler_ReturnsAdvanceError(t *testing.T) {
	stub := &stubWorkflowAdvancer{err: errors.New("db down")}
	handler := &WorkflowJobHandler{logger: (*logging.Logger)(nil), advance: stub}

	require.Error(t, handler.Handle(workflowJobMessage(t, nil, pipelineservice.JobStatusRunning, pipelineservice.JobStatusCompleted)))
}
