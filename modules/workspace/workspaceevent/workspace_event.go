// Package workspaceevent defines the JSON envelope for workspace domain events
// that are published to subscribers via the transactional outbox. Producers
// (storage layer) call EncodeWorkspaceEvent to build the txoutbox message;
// consumers call DecodeWorkspaceEvent to reconstruct the in-process model.
package workspaceevent

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

// WorkspaceEventTopic is the topic name written to txoutbox.Message.Topic for
// every workspace event. Consumers subscribe by this name.
const WorkspaceEventTopic = "workspace.workspace_event"

// String forms of WorkspaceEventType used on the wire. Kept distinct from the
// in-process enum so the wire format is stable across enum re-orderings.
const (
	workspaceEventTypeCreated = "workspace.created"
	workspaceEventTypeUpdated = "workspace.updated"
	workspaceEventTypeDeleted = "workspace.deleted"
)

func eventTypeToString(eventType workspaceservice.WorkspaceEventType) (string, error) {
	switch eventType {
	case workspaceservice.WorkspaceEventTypeCreated:
		return workspaceEventTypeCreated, nil
	case workspaceservice.WorkspaceEventTypeUpdated:
		return workspaceEventTypeUpdated, nil
	case workspaceservice.WorkspaceEventTypeDeleted:
		return workspaceEventTypeDeleted, nil
	default:
		return "", fmt.Errorf("unknown WorkspaceEventType: %d", eventType)
	}
}

func eventTypeFromString(value string) (workspaceservice.WorkspaceEventType, error) {
	switch value {
	case workspaceEventTypeCreated:
		return workspaceservice.WorkspaceEventTypeCreated, nil
	case workspaceEventTypeUpdated:
		return workspaceservice.WorkspaceEventTypeUpdated, nil
	case workspaceEventTypeDeleted:
		return workspaceservice.WorkspaceEventTypeDeleted, nil
	default:
		return 0, fmt.Errorf("unknown WorkspaceEventType string: %q", value)
	}
}

// workspacePayload is the wire format of a Workspace inside an event envelope.
type workspacePayload struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toWorkspacePayload(workspace *workspaceservice.Workspace) workspacePayload {
	return workspacePayload{
		ID:        workspace.ID,
		Name:      workspace.Name,
		Slug:      workspace.Slug,
		CreatedAt: workspace.CreatedAt,
		UpdatedAt: workspace.UpdatedAt,
	}
}

func fromWorkspacePayload(payload workspacePayload) *workspaceservice.Workspace {
	return &workspaceservice.Workspace{
		ID:        payload.ID,
		Name:      payload.Name,
		Slug:      payload.Slug,
		CreatedAt: payload.CreatedAt,
		UpdatedAt: payload.UpdatedAt,
	}
}

type workspaceCreatedPayload struct {
	Workspace workspacePayload `json:"workspace"`
}

type workspaceUpdatedPayload struct {
	OldData workspacePayload `json:"old_data"`
	NewData workspacePayload `json:"new_data"`
}

type workspaceDeletedPayload struct {
	Workspace workspacePayload `json:"workspace"`
}

// EncodeWorkspaceEvent serializes a domain event into its (metadata, data)
// wire form. The returned metadata always includes
// eventutil.MetadataKeyEventType so consumers can dispatch without inspecting
// the body.
func EncodeWorkspaceEvent(event *workspaceservice.WorkspaceEvent) (metadata map[string]string, body []byte, err error) {
	if event == nil {
		return nil, nil, errors.New("event is nil")
	}

	eventTypeStr, err := eventTypeToString(event.EventType)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding event type: %w", err)
	}

	var payload any

	switch data := event.Data.(type) {
	case *workspaceservice.WorkspaceCreatedEventData:
		if data == nil || data.Workspace == nil {
			return nil, nil, errors.New("workspace.created event missing workspace")
		}

		payload = workspaceCreatedPayload{Workspace: toWorkspacePayload(data.Workspace)}
	case *workspaceservice.WorkspaceUpdatedEventData:
		if data == nil || data.OldData == nil || data.NewData == nil {
			return nil, nil, errors.New("workspace.updated event missing old/new workspace")
		}

		payload = workspaceUpdatedPayload{
			OldData: toWorkspacePayload(data.OldData),
			NewData: toWorkspacePayload(data.NewData),
		}
	case *workspaceservice.WorkspaceDeletedEventData:
		if data == nil || data.Workspace == nil {
			return nil, nil, errors.New("workspace.deleted event missing workspace")
		}

		payload = workspaceDeletedPayload{Workspace: toWorkspacePayload(data.Workspace)}
	default:
		return nil, nil, fmt.Errorf("unsupported WorkspaceEventData type: %T", event.Data)
	}

	body, err = json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling event payload: %w", err)
	}

	metadata = map[string]string{
		eventutil.MetadataKeyEventType: eventTypeStr,
	}

	return metadata, body, nil
}

// DecodeWorkspaceEvent rebuilds a domain event from its (metadata, data) wire
// form. Returns an error if metadata is missing the event type, if the type is
// unknown, or if the body does not match the type's expected shape.
func DecodeWorkspaceEvent(metadata map[string]string, body []byte) (*workspaceservice.WorkspaceEvent, error) {
	eventTypeStr, ok := metadata[eventutil.MetadataKeyEventType]
	if !ok {
		return nil, fmt.Errorf("metadata missing %s", eventutil.MetadataKeyEventType)
	}

	eventType, err := eventTypeFromString(eventTypeStr)
	if err != nil {
		return nil, fmt.Errorf("decoding event type: %w", err)
	}

	var data workspaceservice.WorkspaceEventData

	switch eventTypeStr {
	case workspaceEventTypeCreated:
		payload := workspaceCreatedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling created payload: %w", err)
		}

		data = &workspaceservice.WorkspaceCreatedEventData{
			Workspace: fromWorkspacePayload(payload.Workspace),
		}
	case workspaceEventTypeUpdated:
		payload := workspaceUpdatedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling updated payload: %w", err)
		}

		data = &workspaceservice.WorkspaceUpdatedEventData{
			OldData: fromWorkspacePayload(payload.OldData),
			NewData: fromWorkspacePayload(payload.NewData),
		}
	case workspaceEventTypeDeleted:
		payload := workspaceDeletedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling deleted payload: %w", err)
		}

		data = &workspaceservice.WorkspaceDeletedEventData{
			Workspace: fromWorkspacePayload(payload.Workspace),
		}
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventTypeStr)
	}

	return &workspaceservice.WorkspaceEvent{
		EventType: eventType,
		Data:      data,
	}, nil
}

// OrderingKey returns the workspace UUID for the event, suitable for the
// txoutbox ordering key. Events for the same workspace stay in order.
func OrderingKey(event *workspaceservice.WorkspaceEvent) (string, error) {
	if event == nil {
		return "", errors.New("event is nil")
	}

	switch data := event.Data.(type) {
	case *workspaceservice.WorkspaceCreatedEventData:
		if data == nil || data.Workspace == nil {
			return "", errors.New("workspace.created event missing workspace")
		}

		return data.Workspace.ID.String(), nil
	case *workspaceservice.WorkspaceUpdatedEventData:
		if data == nil || data.NewData == nil {
			return "", errors.New("workspace.updated event missing new workspace")
		}

		return data.NewData.ID.String(), nil
	case *workspaceservice.WorkspaceDeletedEventData:
		if data == nil || data.Workspace == nil {
			return "", errors.New("workspace.deleted event missing workspace")
		}

		return data.Workspace.ID.String(), nil
	default:
		return "", fmt.Errorf("unsupported WorkspaceEventData type: %T", event.Data)
	}
}
