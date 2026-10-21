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

type stubRunAdvancer struct {
	runIDs []uuid.UUID
	err    error
}

func (s *stubRunAdvancer) Execute(_ context.Context, runIDs ...uuid.UUID) error {
	s.runIDs = append(s.runIDs, runIDs...)

	return s.err
}

func encodedMessage(t *testing.T, event *pipelineservice.PipelineEvent) *message.Message {
	t.Helper()

	metadata, body, err := pipelineevent.EncodePipelineEvent(event)
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	return msg
}

func runStartedMessage(t *testing.T, runID uuid.UUID) *message.Message {
	t.Helper()

	return encodedMessage(t, &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeWorkflowRunStarted,
		Data:      &pipelineservice.WorkflowRunStartedEventData{WorkspaceID: uuid.New(), WorkflowID: uuid.New(), RunID: runID},
	})
}

func TestWorkflowRunStartedHandler_AdvancesTheRun(t *testing.T) {
	stub := &stubRunAdvancer{}
	handler := &WorkflowRunStartedHandler{logger: (*logging.Logger)(nil), advance: stub}
	runID := uuid.New()

	require.NoError(t, handler.Handle(runStartedMessage(t, runID)))
	assert.Equal(t, []uuid.UUID{runID}, stub.runIDs)
}

func TestWorkflowRunStartedHandler_IgnoresOtherEvents(t *testing.T) {
	stub := &stubRunAdvancer{}
	handler := &WorkflowRunStartedHandler{logger: (*logging.Logger)(nil), advance: stub}

	require.NoError(t, handler.Handle(workflowJobMessage(t, nil, pipelineservice.JobStatusStarting, pipelineservice.JobStatusRunning)))
	assert.Empty(t, stub.runIDs)
}

func TestWorkflowRunStartedHandler_ReturnsAdvanceError(t *testing.T) {
	stub := &stubRunAdvancer{err: errors.New("db down")}
	handler := &WorkflowRunStartedHandler{logger: (*logging.Logger)(nil), advance: stub}

	require.Error(t, handler.Handle(runStartedMessage(t, uuid.New())), "a failed advance is retried by the bus")
}
