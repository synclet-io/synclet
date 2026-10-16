// Package auditevent defines the JSON envelope for audit events that are
// published to subscribers via the transactional outbox. Producers (storage
// layer) call EncodeAuditOutboxEvent to build the txoutbox message; consumers
// call DecodeAuditOutboxEvent to reconstruct the in-process model.
package auditevent

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

// AuditEventTopic is the topic name written to txoutbox.Message.Topic for
// every audit event. Consumers subscribe by this name.
const AuditEventTopic = "audit.audit_event"

// Wire-format strings. Kept distinct from the in-process enum so the wire
// format is stable across enum re-orderings.
const (
	auditOutboxEventTypeRecorded = "audit_event.recorded"
)

func eventTypeToString(eventType auditservice.AuditOutboxEventType) (string, error) {
	switch eventType {
	case auditservice.AuditOutboxEventTypeRecorded:
		return auditOutboxEventTypeRecorded, nil
	default:
		return "", fmt.Errorf("unknown AuditOutboxEventType: %d", eventType)
	}
}

func eventTypeFromString(value string) (auditservice.AuditOutboxEventType, error) {
	switch value {
	case auditOutboxEventTypeRecorded:
		return auditservice.AuditOutboxEventTypeRecorded, nil
	default:
		return 0, fmt.Errorf("unknown AuditOutboxEventType string: %q", value)
	}
}

// auditEventPayload is the wire format of an AuditEvent inside an event
// envelope. Enums travel as their String() names.
type auditEventPayload struct {
	ID            uuid.UUID `json:"id"`
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	ActorType     string    `json:"actor_type"`
	ActorID       uuid.UUID `json:"actor_id"`
	ActorLabel    string    `json:"actor_label"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    uuid.UUID `json:"resource_id"`
	ResourceLabel string    `json:"resource_label"`
	DiffJSON      string    `json:"diff_json"`
	DiffTruncated bool      `json:"diff_truncated"`
	CreatedAt     time.Time `json:"created_at"`
}

func toAuditEventPayload(event *auditservice.AuditEvent) auditEventPayload {
	return auditEventPayload{
		ID:            event.ID,
		WorkspaceID:   event.WorkspaceID,
		ActorType:     event.ActorType.String(),
		ActorID:       event.ActorID,
		ActorLabel:    event.ActorLabel,
		Action:        event.Action.String(),
		ResourceType:  event.ResourceType.String(),
		ResourceID:    event.ResourceID,
		ResourceLabel: event.ResourceLabel,
		DiffJSON:      event.DiffJSON,
		DiffTruncated: event.DiffTruncated,
		CreatedAt:     event.CreatedAt,
	}
}

func fromAuditEventPayload(payload auditEventPayload) (*auditservice.AuditEvent, error) {
	actorType, err := parseEnum[auditservice.ActorType]("ActorType", payload.ActorType)
	if err != nil {
		return nil, fmt.Errorf("decoding actor type: %w", err)
	}

	action, err := parseEnum[auditservice.Action]("Action", payload.Action)
	if err != nil {
		return nil, fmt.Errorf("decoding action: %w", err)
	}

	resourceType, err := parseEnum[auditservice.ResourceType]("ResourceType", payload.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("decoding resource type: %w", err)
	}

	return &auditservice.AuditEvent{
		ID:            payload.ID,
		WorkspaceID:   payload.WorkspaceID,
		ActorType:     actorType,
		ActorID:       payload.ActorID,
		ActorLabel:    payload.ActorLabel,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    payload.ResourceID,
		ResourceLabel: payload.ResourceLabel,
		DiffJSON:      payload.DiffJSON,
		DiffTruncated: payload.DiffTruncated,
		CreatedAt:     payload.CreatedAt,
	}, nil
}

type namedEnum interface {
	~byte
	IsValid() bool
	String() string
}

// parseEnum maps a String() name back to its enum constant by scanning every
// valid value of the enum.
func parseEnum[T namedEnum](name, value string) (T, error) {
	for candidate := T(1); candidate.IsValid(); candidate++ {
		if candidate.String() == value {
			return candidate, nil
		}
	}

	return 0, fmt.Errorf("unknown %s: %q", name, value)
}

type auditEventRecordedPayload struct {
	Event auditEventPayload `json:"event"`
}

// RecordedEvent returns the audit event carried by a recorded outbox event.
func RecordedEvent(event *auditservice.AuditOutboxEvent) (*auditservice.AuditEvent, error) {
	if event == nil {
		return nil, errors.New("event is nil")
	}

	switch data := event.Data.(type) {
	case *auditservice.AuditEventRecordedData:
		if data == nil || data.Event == nil {
			return nil, errors.New("audit_event.recorded event missing audit event")
		}

		return data.Event, nil
	default:
		return nil, fmt.Errorf("unsupported AuditOutboxEventData type: %T", event.Data)
	}
}

// EncodeAuditOutboxEvent serializes a domain event into its (metadata, data)
// wire form. The returned metadata always includes
// eventutil.MetadataKeyEventType so consumers can dispatch without inspecting
// the body.
func EncodeAuditOutboxEvent(event *auditservice.AuditOutboxEvent) (metadata map[string]string, body []byte, err error) {
	if event == nil {
		return nil, nil, errors.New("event is nil")
	}

	eventTypeStr, err := eventTypeToString(event.EventType)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding event type: %w", err)
	}

	recorded, err := RecordedEvent(event)
	if err != nil {
		return nil, nil, err
	}

	body, err = json.Marshal(auditEventRecordedPayload{Event: toAuditEventPayload(recorded)})
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling event payload: %w", err)
	}

	metadata = map[string]string{
		eventutil.MetadataKeyEventType: eventTypeStr,
	}

	return metadata, body, nil
}

// DecodeAuditOutboxEvent rebuilds a domain event from its (metadata, data)
// wire form.
func DecodeAuditOutboxEvent(metadata map[string]string, body []byte) (*auditservice.AuditOutboxEvent, error) {
	eventTypeStr, ok := metadata[eventutil.MetadataKeyEventType]
	if !ok {
		return nil, fmt.Errorf("metadata missing %s", eventutil.MetadataKeyEventType)
	}

	eventType, err := eventTypeFromString(eventTypeStr)
	if err != nil {
		return nil, fmt.Errorf("decoding event type: %w", err)
	}

	var data auditservice.AuditOutboxEventData

	switch eventTypeStr {
	case auditOutboxEventTypeRecorded:
		payload := auditEventRecordedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling audit_event.recorded payload: %w", err)
		}

		recorded, err := fromAuditEventPayload(payload.Event)
		if err != nil {
			return nil, fmt.Errorf("decoding audit event: %w", err)
		}

		data = &auditservice.AuditEventRecordedData{Event: recorded}
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventTypeStr)
	}

	return &auditservice.AuditOutboxEvent{
		EventType: eventType,
		Data:      data,
	}, nil
}

// OrderingKey returns the workspace UUID for the event, suitable for the
// txoutbox ordering key. Events for the same workspace stay in order.
func OrderingKey(event *auditservice.AuditOutboxEvent) (string, error) {
	recorded, err := RecordedEvent(event)
	if err != nil {
		return "", err
	}

	return recorded.WorkspaceID.String(), nil
}

// IdempotencyKey returns the audit event UUID, so every delivery of the same
// audit event carries the same key.
func IdempotencyKey(event *auditservice.AuditOutboxEvent) (string, error) {
	recorded, err := RecordedEvent(event)
	if err != nil {
		return "", err
	}

	return recorded.ID.String(), nil
}
