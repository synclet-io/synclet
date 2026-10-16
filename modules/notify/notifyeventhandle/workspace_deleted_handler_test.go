package notifyeventhandle

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/workspace/workspaceevent"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

type stubPurger struct {
	called []uuid.UUID
	err    error
}

func (s *stubPurger) Execute(_ context.Context, workspaceID uuid.UUID) error {
	s.called = append(s.called, workspaceID)

	return s.err
}

func newWorkspaceDeletedHandler(stub *stubPurger) *WorkspaceDeletedHandler {
	return &WorkspaceDeletedHandler{logger: &logging.Logger{Delegate: noopLogDelegate{}}, purge: stub}
}

func workspaceMessage(t *testing.T, event *workspaceservice.WorkspaceEvent) *message.Message {
	t.Helper()

	metadata, body, err := workspaceevent.EncodeWorkspaceEvent(event)
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	return msg
}

func workspaceDeletedMessage(t *testing.T, workspaceID uuid.UUID) *message.Message {
	t.Helper()

	return workspaceMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeDeleted,
		Data:      &workspaceservice.WorkspaceDeletedEventData{Workspace: &workspaceservice.Workspace{ID: workspaceID}},
	})
}

func TestWorkspaceDeletedHandler_PurgesWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	stub := &stubPurger{}

	require.NoError(t, newWorkspaceDeletedHandler(stub).Handle(workspaceDeletedMessage(t, workspaceID)))
	assert.Equal(t, []uuid.UUID{workspaceID}, stub.called)
}

func TestWorkspaceDeletedHandler_IgnoresOtherEvents(t *testing.T) {
	stub := &stubPurger{}

	msg := workspaceMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data:      &workspaceservice.WorkspaceCreatedEventData{Workspace: &workspaceservice.Workspace{ID: uuid.New()}},
	})

	require.NoError(t, newWorkspaceDeletedHandler(stub).Handle(msg))
	assert.Empty(t, stub.called)
}

func TestWorkspaceDeletedHandler_DecodeFailureIsAcked(t *testing.T) {
	stub := &stubPurger{}

	require.NoError(t, newWorkspaceDeletedHandler(stub).Handle(message.NewMessage(uuid.NewString(), []byte("{}"))))
	assert.Empty(t, stub.called)
}

func TestWorkspaceDeletedHandler_PurgeErrorIsRetried(t *testing.T) {
	stub := &stubPurger{err: errors.New("db down")}

	require.Error(t, newWorkspaceDeletedHandler(stub).Handle(workspaceDeletedMessage(t, uuid.New())))
}

func TestWorkspaceDeletedHandler_Metadata(t *testing.T) {
	handler := newWorkspaceDeletedHandler(&stubPurger{})

	assert.Equal(t, "notify_workspace_deleted", handler.Name())
	assert.Equal(t, workspaceevent.WorkspaceEventTopic, handler.Topic())
}
