package auditevent

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

func newAuditEvent() *auditservice.AuditEvent {
	now := time.Now().UTC().Truncate(time.Microsecond)

	return &auditservice.AuditEvent{
		ID:            uuid.New(),
		WorkspaceID:   uuid.New(),
		ActorType:     auditservice.ActorTypeUser,
		ActorID:       uuid.New(),
		ActorLabel:    "alice@example.com",
		Action:        auditservice.ActionUpdate,
		ResourceType:  auditservice.ResourceTypeSource,
		ResourceID:    uuid.New(),
		ResourceLabel: "postgres-prod",
		DiffJSON:      `[{"path":"name","before":"a","after":"b"}]`,
		DiffTruncated: true,
		CreatedAt:     now,
	}
}

func newRecordedEvent(event *auditservice.AuditEvent) *auditservice.AuditOutboxEvent {
	return &auditservice.AuditOutboxEvent{
		EventType: auditservice.AuditOutboxEventTypeRecorded,
		Data:      &auditservice.AuditEventRecordedData{Event: event},
	}
}

func TestEncodeDecode_RecordedRoundTrip(t *testing.T) {
	event := newRecordedEvent(newAuditEvent())

	metadata, body, err := EncodeAuditOutboxEvent(event)
	require.NoError(t, err)
	assert.Equal(t, "audit_event.recorded", metadata[eventutil.MetadataKeyEventType])
	assert.NotEmpty(t, body)

	decoded, err := DecodeAuditOutboxEvent(metadata, body)
	require.NoError(t, err)
	require.True(t, event.Equals(decoded), "round-trip must preserve event equality")
}

func TestEncodeDecode_EveryEnumValueRoundTrips(t *testing.T) {
	for actorType := auditservice.ActorType(1); actorType.IsValid(); actorType++ {
		for action := auditservice.Action(1); action.IsValid(); action++ {
			for resourceType := auditservice.ResourceType(1); resourceType.IsValid(); resourceType++ {
				audit := newAuditEvent()
				audit.ActorType = actorType
				audit.Action = action
				audit.ResourceType = resourceType
				event := newRecordedEvent(audit)

				metadata, body, err := EncodeAuditOutboxEvent(event)
				require.NoError(t, err)

				decoded, err := DecodeAuditOutboxEvent(metadata, body)
				require.NoError(t, err)
				require.True(t, event.Equals(decoded), "%s/%s/%s", actorType, action, resourceType)
			}
		}
	}
}

func TestOrderingKey_IsWorkspaceID(t *testing.T) {
	audit := newAuditEvent()

	got, err := OrderingKey(newRecordedEvent(audit))
	require.NoError(t, err)
	assert.Equal(t, audit.WorkspaceID.String(), got)
}

func TestIdempotencyKey_IsAuditEventID(t *testing.T) {
	audit := newAuditEvent()

	got, err := IdempotencyKey(newRecordedEvent(audit))
	require.NoError(t, err)
	assert.Equal(t, audit.ID.String(), got)
}

func TestRecordedEvent_ReturnsCarriedEvent(t *testing.T) {
	audit := newAuditEvent()

	got, err := RecordedEvent(newRecordedEvent(audit))
	require.NoError(t, err)
	assert.Same(t, audit, got)
}

func TestOrderingKey_NilEvent(t *testing.T) {
	_, err := OrderingKey(nil)
	require.Error(t, err)
}

func TestIdempotencyKey_MissingAuditEvent(t *testing.T) {
	_, err := IdempotencyKey(newRecordedEvent(nil))
	require.Error(t, err)
}

func TestEncode_NilEvent(t *testing.T) {
	_, _, err := EncodeAuditOutboxEvent(nil)
	require.Error(t, err)
}

func TestEncode_MissingAuditEvent(t *testing.T) {
	_, _, err := EncodeAuditOutboxEvent(newRecordedEvent(nil))
	require.Error(t, err)
}

func TestEncode_NilData(t *testing.T) {
	_, _, err := EncodeAuditOutboxEvent(&auditservice.AuditOutboxEvent{
		EventType: auditservice.AuditOutboxEventTypeRecorded,
	})
	require.Error(t, err)
}

func TestEncode_UnsupportedEventType(t *testing.T) {
	event := newRecordedEvent(newAuditEvent())
	event.EventType = auditservice.AuditOutboxEventType(99)

	_, _, err := EncodeAuditOutboxEvent(event)
	require.Error(t, err)
}

func TestDecode_MissingMetadataKey(t *testing.T) {
	_, err := DecodeAuditOutboxEvent(map[string]string{}, []byte("{}"))
	require.Error(t, err)
}

func TestDecode_UnknownEventType(t *testing.T) {
	_, err := DecodeAuditOutboxEvent(map[string]string{
		eventutil.MetadataKeyEventType: "audit_event.unknown",
	}, []byte("{}"))
	require.Error(t, err)
}

func TestDecode_BadJSON(t *testing.T) {
	_, err := DecodeAuditOutboxEvent(map[string]string{
		eventutil.MetadataKeyEventType: "audit_event.recorded",
	}, []byte("not-json"))
	require.Error(t, err)
}

func TestDecode_UnknownEnumName(t *testing.T) {
	for _, body := range []string{
		`{"event":{"actor_type":"Ghost","action":"Update","resource_type":"Source"}}`,
		`{"event":{"actor_type":"User","action":"Explode","resource_type":"Source"}}`,
		`{"event":{"actor_type":"User","action":"Update","resource_type":"Teapot"}}`,
	} {
		_, err := DecodeAuditOutboxEvent(map[string]string{
			eventutil.MetadataKeyEventType: "audit_event.recorded",
		}, []byte(body))
		require.Error(t, err, body)
	}
}
