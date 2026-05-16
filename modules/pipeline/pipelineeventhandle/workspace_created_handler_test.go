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

	"github.com/synclet-io/synclet/modules/workspace/workspaceevent"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

type stubCreator struct {
	called      []uuid.UUID
	executeErr  error
	executeCall int
}

func (s *stubCreator) Execute(_ context.Context, workspaceID uuid.UUID) error {
	s.executeCall++
	s.called = append(s.called, workspaceID)

	return s.executeErr
}

func newHandlerWithStub(stub *stubCreator) *WorkspaceCreatedHandler {
	return &WorkspaceCreatedHandler{
		logger:      (*logging.Logger)(nil),
		createRepos: stub,
	}
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

func TestHandle_CreatedEventSeedsRegistries(t *testing.T) {
	workspaceID := uuid.New()
	stub := &stubCreator{}
	handler := newHandlerWithStub(stub)

	msg := encodeMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data: &workspaceservice.WorkspaceCreatedEventData{
			Workspace: &workspaceservice.Workspace{ID: workspaceID, Name: "n", Slug: "n"},
		},
	})

	require.NoError(t, handler.Handle(msg))
	require.Equal(t, 1, stub.executeCall)
	assert.Equal(t, workspaceID, stub.called[0])
}

func TestHandle_UpdatedEventIgnored(t *testing.T) {
	stub := &stubCreator{}
	handler := newHandlerWithStub(stub)

	ws := &workspaceservice.Workspace{ID: uuid.New(), Name: "n", Slug: "n"}
	msg := encodeMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeUpdated,
		Data:      &workspaceservice.WorkspaceUpdatedEventData{OldData: ws, NewData: ws},
	})

	require.NoError(t, handler.Handle(msg))
	assert.Equal(t, 0, stub.executeCall, "updated events must not trigger the seeder")
}

func TestHandle_DeletedEventIgnored(t *testing.T) {
	stub := &stubCreator{}
	handler := newHandlerWithStub(stub)

	msg := encodeMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeDeleted,
		Data:      &workspaceservice.WorkspaceDeletedEventData{Workspace: &workspaceservice.Workspace{ID: uuid.New()}},
	})

	require.NoError(t, handler.Handle(msg))
	assert.Equal(t, 0, stub.executeCall)
}

func TestHandle_DecodeFailureIsAcked(t *testing.T) {
	stub := &stubCreator{}
	handler := newHandlerWithStub(stub)

	// Missing event-type metadata triggers a decode error.
	msg := message.NewMessage(uuid.NewString(), []byte("{}"))
	require.NoError(t, handler.Handle(msg), "decode failure must be acked, not retried")
	assert.Equal(t, 0, stub.executeCall)
}

func TestHandle_UseCaseErrorPropagated(t *testing.T) {
	stub := &stubCreator{executeErr: errors.New("db down")}
	handler := newHandlerWithStub(stub)

	msg := encodeMessage(t, &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data: &workspaceservice.WorkspaceCreatedEventData{
			Workspace: &workspaceservice.Workspace{ID: uuid.New(), Name: "n", Slug: "n"},
		},
	})

	err := handler.Handle(msg)
	require.Error(t, err, "storage failures must nack so Watermill retries")
}

func TestHandlerMetadata(t *testing.T) {
	handler := newHandlerWithStub(&stubCreator{})
	assert.NotEmpty(t, handler.Name())
	assert.Equal(t, workspaceevent.WorkspaceEventTopic, handler.Topic())
}
