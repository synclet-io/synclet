package workspaceevent

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

func newWorkspace(name string) *workspaceservice.Workspace {
	now := time.Now().UTC().Truncate(time.Microsecond)

	return &workspaceservice.Workspace{
		ID:        uuid.New(),
		Name:      name,
		Slug:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestEncodeDecode_CreatedRoundTrip(t *testing.T) {
	ws := newWorkspace("acme")
	event := &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data:      &workspaceservice.WorkspaceCreatedEventData{Workspace: ws},
	}

	metadata, body, err := EncodeWorkspaceEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "workspace.created", metadata[eventutil.MetadataKeyEventType])
	assert.NotEmpty(t, body)

	decoded, err := DecodeWorkspaceEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded), "round-trip must preserve event equality")
}

func TestEncodeDecode_UpdatedRoundTrip(t *testing.T) {
	oldWs := newWorkspace("acme")
	newWs := oldWs.Copy()
	newWs.Name = "Acme Renamed"
	event := &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeUpdated,
		Data: &workspaceservice.WorkspaceUpdatedEventData{
			OldData: oldWs,
			NewData: &newWs,
		},
	}

	metadata, body, err := EncodeWorkspaceEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "workspace.updated", metadata[eventutil.MetadataKeyEventType])

	decoded, err := DecodeWorkspaceEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded))
}

func TestEncodeDecode_DeletedRoundTrip(t *testing.T) {
	ws := newWorkspace("acme")
	event := &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeDeleted,
		Data:      &workspaceservice.WorkspaceDeletedEventData{Workspace: ws},
	}

	metadata, body, err := EncodeWorkspaceEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "workspace.deleted", metadata[eventutil.MetadataKeyEventType])

	decoded, err := DecodeWorkspaceEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded))
}

func TestEncode_NilEvent(t *testing.T) {
	_, _, err := EncodeWorkspaceEvent(nil)
	require.Error(t, err)
}

func TestEncode_MissingWorkspaceData(t *testing.T) {
	event := &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data:      &workspaceservice.WorkspaceCreatedEventData{},
	}
	_, _, err := EncodeWorkspaceEvent(event)
	require.Error(t, err)
}

func TestDecode_MissingMetadataKey(t *testing.T) {
	_, err := DecodeWorkspaceEvent(map[string]string{}, []byte("{}"))
	require.Error(t, err)
}

func TestDecode_UnknownEventType(t *testing.T) {
	_, err := DecodeWorkspaceEvent(map[string]string{
		eventutil.MetadataKeyEventType: "workspace.unknown",
	}, []byte("{}"))
	require.Error(t, err)
}

func TestDecode_BadJSON(t *testing.T) {
	_, err := DecodeWorkspaceEvent(map[string]string{
		eventutil.MetadataKeyEventType: "workspace.created",
	}, []byte("not-json"))
	require.Error(t, err)
}

func TestOrderingKey_CreatedEvent(t *testing.T) {
	ws := newWorkspace("a")
	event := &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeCreated,
		Data:      &workspaceservice.WorkspaceCreatedEventData{Workspace: ws},
	}
	got, err := OrderingKey(event)
	require.NoError(t, err)
	assert.Equal(t, ws.ID.String(), got)
}

func TestOrderingKey_UpdatedUsesNewID(t *testing.T) {
	oldWs := newWorkspace("a")
	newWs := newWorkspace("b")
	event := &workspaceservice.WorkspaceEvent{
		EventType: workspaceservice.WorkspaceEventTypeUpdated,
		Data:      &workspaceservice.WorkspaceUpdatedEventData{OldData: oldWs, NewData: newWs},
	}
	got, err := OrderingKey(event)
	require.NoError(t, err)
	assert.Equal(t, newWs.ID.String(), got)
}

func TestOrderingKey_NilEvent(t *testing.T) {
	_, err := OrderingKey(nil)
	require.Error(t, err)
}
