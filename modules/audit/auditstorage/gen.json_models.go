package auditstorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	time "time"

	uuid "github.com/google/uuid"

	auditservice "github.com/synclet-io/synclet/modules/audit/auditservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonAuditEvent struct {
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

func (a *jsonAuditEvent) Scan(value any) error {
	return json.Unmarshal(value.([]byte), a)
}

func (a jsonAuditEvent) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func convertAuditEventToJsonModel(src *auditservice.AuditEvent) (*jsonAuditEvent, error) {
	result := &jsonAuditEvent{}
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

func convertAuditEventFromJsonModel(src *jsonAuditEvent) (*auditservice.AuditEvent, error) {
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
