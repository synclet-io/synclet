package auditstorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	fmt "fmt"
	time "time"

	uuid "github.com/google/uuid"
	errors "github.com/pkg/errors"

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

type jsonAuditRetentionSetting struct {
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	RetentionDays int       `json:"retention_days"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (a *jsonAuditRetentionSetting) Scan(value any) error {
	return json.Unmarshal(value.([]byte), a)
}

func (a jsonAuditRetentionSetting) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func convertAuditRetentionSettingToJsonModel(src *auditservice.AuditRetentionSetting) (*jsonAuditRetentionSetting, error) {
	result := &jsonAuditRetentionSetting{}
	result.WorkspaceID = src.WorkspaceID
	result.RetentionDays = src.RetentionDays
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertAuditRetentionSettingFromJsonModel(src *jsonAuditRetentionSetting) (*auditservice.AuditRetentionSetting, error) {
	result := &auditservice.AuditRetentionSetting{}
	result.WorkspaceID = src.WorkspaceID
	result.RetentionDays = src.RetentionDays
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonAuditOutboxEvent struct {
	EventType string                    `json:"event_type"`
	Data      *jsonAuditOutboxEventData `json:"data"`
}

func (a *jsonAuditOutboxEvent) Scan(value any) error {
	return json.Unmarshal(value.([]byte), a)
}

func (a jsonAuditOutboxEvent) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func convertAuditOutboxEventToJsonModel(src *auditservice.AuditOutboxEvent) (*jsonAuditOutboxEvent, error) {
	result := &jsonAuditOutboxEvent{}
	tmp, err := convertAuditOutboxEventTypeToDB(src.EventType)
	if err != nil {
		return nil, err
	}
	result.EventType = tmp
	tmp1, err := convertAuditOutboxEventDataToDB(src.Data)
	if err != nil {
		return nil, err
	}
	result.Data = tmp1
	return result, nil
}

func convertAuditOutboxEventFromJsonModel(src *jsonAuditOutboxEvent) (*auditservice.AuditOutboxEvent, error) {
	result := &auditservice.AuditOutboxEvent{}
	tmp2, err := convertAuditOutboxEventTypeFromDB(src.EventType)
	if err != nil {
		return nil, err
	}
	result.EventType = tmp2
	tmp3, err := convertAuditOutboxEventDataFromDB(src.Data)
	if err != nil {
		return nil, fmt.Errorf("convert AuditOutboxEventData to service type: %w", err)
	}
	result.Data = tmp3
	return result, nil
}

type jsonAuditEventRecordedData struct {
	Event *jsonAuditEvent `json:"event"`
}

func (a *jsonAuditEventRecordedData) Scan(value any) error {
	return json.Unmarshal(value.([]byte), a)
}

func (a jsonAuditEventRecordedData) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func convertAuditEventRecordedDataToJsonModel(src *auditservice.AuditEventRecordedData) (*jsonAuditEventRecordedData, error) {
	result := &jsonAuditEventRecordedData{}
	if src.Event != nil {
		tmp, err := convertAuditEventToJsonModel(src.Event)
		if err != nil {
			return nil, errors.Wrap(err, "convert AuditEvent to db")
		}
		result.Event = tmp
	} else {
		result.Event = nil
	}
	return result, nil
}

func convertAuditEventRecordedDataFromJsonModel(src *jsonAuditEventRecordedData) (*auditservice.AuditEventRecordedData, error) {
	result := &auditservice.AuditEventRecordedData{}
	if src.Event != nil {
		tmp1, err := convertAuditEventFromJsonModel(src.Event)
		if err != nil {
			return nil, err
		}
		result.Event = tmp1
	} else {
		result.Event = nil
	}
	return result, nil
}
