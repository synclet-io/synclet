package autheventhandle

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

func newHandler(stub *stubPurger) *WorkspaceDeletedHandler {
	return &WorkspaceDeletedHandler{logger: (*logging.Logger)(nil), purge: stub}
}

func encodeMessage(t *testing.T, event *workspaceservice.WorkspaceEvent) *message.Message {
	t.Helper()

	metadata, body, err := workspaceevent.EncodeWorkspaceEvent(event)
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	return msg
}

func deletedMessage(t *testing.T, workspaceID uuid.UUID) *message.Message {
	t.Helper()

	return encodeMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeDeleted,
		Data:      &workspaceservice.WorkspaceDeletedEventData{Workspace: &workspaceservice.Workspace{ID: workspaceID}},
	})
}

func TestHandle_DeletedEventPurgesAPIKeys(t *testing.T) {
	workspaceID := uuid.New()
	stub := &stubPurger{}

	require.NoError(t, newHandler(stub).Handle(deletedMessage(t, workspaceID)))
	assert.Equal(t, []uuid.UUID{workspaceID}, stub.called)
}

func TestHandle_OtherEventsIgnored(t *testing.T) {
	stub := &stubPurger{}

	msg := encodeMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data:      &workspaceservice.WorkspaceCreatedEventData{Workspace: &workspaceservice.Workspace{ID: uuid.New()}},
	})

	require.NoError(t, newHandler(stub).Handle(msg))
	assert.Empty(t, stub.called)
}

func TestHandle_DecodeFailureIsAcked(t *testing.T) {
	stub := &stubPurger{}

	require.NoError(t, newHandler(stub).Handle(message.NewMessage(uuid.NewString(), []byte("{}"))))
	assert.Empty(t, stub.called)
}

func TestHandle_PurgeErrorIsRetried(t *testing.T) {
	stub := &stubPurger{err: errors.New("db down")}

	require.Error(t, newHandler(stub).Handle(deletedMessage(t, uuid.New())))
}

func TestHandlerMetadata(t *testing.T) {
	handler := newHandler(&stubPurger{})

	assert.Equal(t, "auth_workspace_deleted", handler.Name())
	assert.Equal(t, workspaceevent.WorkspaceEventTopic, handler.Topic())
}
