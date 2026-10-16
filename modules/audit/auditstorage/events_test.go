package auditstorage

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/audit/auditevent"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

func TestBuildAuditOutboxEventMessage(t *testing.T) {
	audit := &auditservice.AuditEvent{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		ActorType:    auditservice.ActorTypeSystem,
		Action:       auditservice.ActionCreate,
		ResourceType: auditservice.ResourceTypeConnection,
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	event := &auditservice.AuditOutboxEvent{
		EventType: auditservice.AuditOutboxEventTypeRecorded,
		Data:      &auditservice.AuditEventRecordedData{Event: audit},
	}

	msg, err := buildAuditOutboxEventMessage(event)
	require.NoError(t, err)

	assert.Equal(t, auditevent.AuditEventTopic, msg.Topic)
	assert.Equal(t, audit.WorkspaceID.String(), msg.OrderingKey)
	assert.Equal(t, audit.ID.String(), msg.IdempotencyKey)
	assert.Equal(t, "audit_event.recorded", msg.Metadata[eventutil.MetadataKeyEventType])

	decoded, err := auditevent.DecodeAuditOutboxEvent(msg.Metadata, msg.Data)
	require.NoError(t, err)
	assert.True(t, event.Equals(decoded))
}

func TestBuildAuditOutboxEventMessage_MissingAuditEvent(t *testing.T) {
	_, err := buildAuditOutboxEventMessage(&auditservice.AuditOutboxEvent{
		EventType: auditservice.AuditOutboxEventTypeRecorded,
		Data:      &auditservice.AuditEventRecordedData{},
	})
	require.Error(t, err)
}
