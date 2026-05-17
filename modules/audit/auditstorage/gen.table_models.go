package auditstorage

import (
	time "time"

	uuid "github.com/google/uuid"

	auditservice "github.com/synclet-io/synclet/modules/audit/auditservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbAuditEvent struct {
	ID            uuid.UUID `gorm:"column:id;"`
	WorkspaceID   uuid.UUID `gorm:"column:workspace_id;"`
	ActorType     string    `gorm:"column:actor_type;type:text;"`
	ActorID       uuid.UUID `gorm:"column:actor_id;"`
	ActorLabel    string    `gorm:"column:actor_label;type:text;"`
	Action        string    `gorm:"column:action;type:text;"`
	ResourceType  string    `gorm:"column:resource_type;type:text;"`
	ResourceID    uuid.UUID `gorm:"column:resource_id;"`
	ResourceLabel string    `gorm:"column:resource_label;type:text;"`
	DiffJSON      string    `gorm:"column:diff_json;type:text;"`
	DiffTruncated bool      `gorm:"column:diff_truncated;"`
	CreatedAt     time.Time `gorm:"column:created_at;"`
}

func convertAuditEventToDB(src *auditservice.AuditEvent) (*dbAuditEvent, error) {
	result := &dbAuditEvent{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	tmp2, err := convertActorTypeToDB(src.ActorType)
	if err != nil {
		return nil, err
	}
	result.ActorType = tmp2
	result.ActorID = src.ActorID
	result.ActorLabel = src.ActorLabel
	tmp5, err := convertActionToDB(src.Action)
	if err != nil {
		return nil, err
	}
	result.Action = tmp5
	tmp6, err := convertResourceTypeToDB(src.ResourceType)
	if err != nil {
		return nil, err
	}
	result.ResourceType = tmp6
	result.ResourceID = src.ResourceID
	result.ResourceLabel = src.ResourceLabel
	result.DiffJSON = src.DiffJSON
	result.DiffTruncated = src.DiffTruncated
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertAuditEventFromDB(src *dbAuditEvent) (*auditservice.AuditEvent, error) {
	result := &auditservice.AuditEvent{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	tmp14, err := convertActorTypeFromDB(src.ActorType)
	if err != nil {
		return nil, err
	}
	result.ActorType = tmp14
	result.ActorID = src.ActorID
	result.ActorLabel = src.ActorLabel
	tmp17, err := convertActionFromDB(src.Action)
	if err != nil {
		return nil, err
	}
	result.Action = tmp17
	tmp18, err := convertResourceTypeFromDB(src.ResourceType)
	if err != nil {
		return nil, err
	}
	result.ResourceType = tmp18
	result.ResourceID = src.ResourceID
	result.ResourceLabel = src.ResourceLabel
	result.DiffJSON = src.DiffJSON
	result.DiffTruncated = src.DiffTruncated
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbAuditEvent) TableName() string {
	return "audit.audit_events"
}
