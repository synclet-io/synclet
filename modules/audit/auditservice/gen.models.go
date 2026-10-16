package auditservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

//sumtype:decl
type AuditOutboxEventData interface {
	isAuditOutboxEventData()
	AuditOutboxEventDataEquals(AuditOutboxEventData) bool
	// user code 'AuditOutboxEventData methods'
	// end user code 'AuditOutboxEventData methods'
}

func (*AuditEventRecordedData) isAuditOutboxEventData() {}
func (a *AuditEventRecordedData) AuditOutboxEventDataEquals(to AuditOutboxEventData) bool {
	if (a == nil) != (to == nil) {
		return false
	}
	if a == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*AuditEventRecordedData)
	if !ok {
		return false
	}

	return a.Equals(toTyped)
}

func copyAuditOutboxEventData(val AuditOutboxEventData) AuditOutboxEventData {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case *AuditEventRecordedData:
		valCopy := val.Copy()
		return &valCopy
	}
	panic("called copyAuditOutboxEventData with invalid type")
}

type AuditEventField byte

const (
	AuditEventFieldID AuditEventField = iota + 1
	AuditEventFieldWorkspaceID
	AuditEventFieldActorType
	AuditEventFieldActorID
	AuditEventFieldActorLabel
	AuditEventFieldAction
	AuditEventFieldResourceType
	AuditEventFieldResourceID
	AuditEventFieldResourceLabel
	AuditEventFieldDiffJSON
	AuditEventFieldDiffTruncated
	AuditEventFieldCreatedAt
)

type AuditEventFilter struct {
	ID           filter.Filter[uuid.UUID]
	WorkspaceID  filter.Filter[uuid.UUID]
	ActorType    filter.Filter[ActorType]
	ActorID      filter.Filter[uuid.UUID]
	Action       filter.Filter[Action]
	ResourceType filter.Filter[ResourceType]
	ResourceID   filter.Filter[uuid.UUID]
	CreatedAt    filter.Filter[time.Time]
	Or           []*AuditEventFilter
	And          []*AuditEventFilter
}
type AuditEventOrder order.Order[AuditEventField]

type AuditEvent struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	ActorType     ActorType
	ActorID       uuid.UUID
	ActorLabel    string
	Action        Action
	ResourceType  ResourceType
	ResourceID    uuid.UUID
	ResourceLabel string
	DiffJSON      string
	DiffTruncated bool
	CreatedAt     time.Time
}

// user code 'AuditEvent methods'
// end user code 'AuditEvent methods'

func (a *AuditEvent) Copy() AuditEvent {
	var result AuditEvent
	result.ID = a.ID
	result.WorkspaceID = a.WorkspaceID
	result.ActorType = a.ActorType // enum
	result.ActorID = a.ActorID
	result.ActorLabel = a.ActorLabel
	result.Action = a.Action             // enum
	result.ResourceType = a.ResourceType // enum
	result.ResourceID = a.ResourceID
	result.ResourceLabel = a.ResourceLabel
	result.DiffJSON = a.DiffJSON
	result.DiffTruncated = a.DiffTruncated
	result.CreatedAt = a.CreatedAt

	return result
}
func (a *AuditEvent) Equals(to *AuditEvent) bool {
	if (a == nil) != (to == nil) {
		return false
	}
	if a == nil && to == nil {
		return true
	}
	if a.ID != to.ID {
		return false
	}
	if a.WorkspaceID != to.WorkspaceID {
		return false
	}
	if a.ActorType != to.ActorType {
		return false
	}
	if a.ActorID != to.ActorID {
		return false
	}
	if a.ActorLabel != to.ActorLabel {
		return false
	}
	if a.Action != to.Action {
		return false
	}
	if a.ResourceType != to.ResourceType {
		return false
	}
	if a.ResourceID != to.ResourceID {
		return false
	}
	if a.ResourceLabel != to.ResourceLabel {
		return false
	}
	if a.DiffJSON != to.DiffJSON {
		return false
	}
	if a.DiffTruncated != to.DiffTruncated {
		return false
	}
	if a.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type AuditRetentionSettingField byte

const (
	AuditRetentionSettingFieldWorkspaceID AuditRetentionSettingField = iota + 1
	AuditRetentionSettingFieldRetentionDays
	AuditRetentionSettingFieldUpdatedAt
)

type AuditRetentionSettingFilter struct {
	WorkspaceID filter.Filter[uuid.UUID]
	Or          []*AuditRetentionSettingFilter
	And         []*AuditRetentionSettingFilter
}
type AuditRetentionSettingOrder order.Order[AuditRetentionSettingField]

type AuditRetentionSetting struct {
	WorkspaceID   uuid.UUID
	RetentionDays int
	UpdatedAt     time.Time
}

// user code 'AuditRetentionSetting methods'
// end user code 'AuditRetentionSetting methods'

func (a *AuditRetentionSetting) Copy() AuditRetentionSetting {
	var result AuditRetentionSetting
	result.WorkspaceID = a.WorkspaceID
	result.RetentionDays = a.RetentionDays
	result.UpdatedAt = a.UpdatedAt

	return result
}
func (a *AuditRetentionSetting) Equals(to *AuditRetentionSetting) bool {
	if (a == nil) != (to == nil) {
		return false
	}
	if a == nil && to == nil {
		return true
	}
	if a.WorkspaceID != to.WorkspaceID {
		return false
	}
	if a.RetentionDays != to.RetentionDays {
		return false
	}
	if a.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type AuditOutboxEventField byte

const (
	AuditOutboxEventFieldEventType AuditOutboxEventField = iota + 1
	AuditOutboxEventFieldData
)

type AuditOutboxEventFilter struct {
	Or  []*AuditOutboxEventFilter
	And []*AuditOutboxEventFilter
}
type AuditOutboxEventOrder order.Order[AuditOutboxEventField]

type AuditOutboxEvent struct {
	EventType AuditOutboxEventType
	Data      AuditOutboxEventData
}

// user code 'AuditOutboxEvent methods'
// end user code 'AuditOutboxEvent methods'

func (a *AuditOutboxEvent) Copy() AuditOutboxEvent {
	var result AuditOutboxEvent
	result.EventType = a.EventType // enum
	result.Data = copyAuditOutboxEventData(a.Data)

	return result
}
func (a *AuditOutboxEvent) Equals(to *AuditOutboxEvent) bool {
	if (a == nil) != (to == nil) {
		return false
	}
	if a == nil && to == nil {
		return true
	}
	if a.EventType != to.EventType {
		return false
	}
	if !a.Data.AuditOutboxEventDataEquals(to.Data) {
		return false
	}

	return true
}

type AuditEventRecordedDataField byte

const (
	AuditEventRecordedDataFieldEvent AuditEventRecordedDataField = iota + 1
)

type AuditEventRecordedDataFilter struct {
	Or  []*AuditEventRecordedDataFilter
	And []*AuditEventRecordedDataFilter
}
type AuditEventRecordedDataOrder order.Order[AuditEventRecordedDataField]

type AuditEventRecordedData struct {
	Event *AuditEvent
}

// user code 'AuditEventRecordedData methods'
// end user code 'AuditEventRecordedData methods'

func (a *AuditEventRecordedData) Copy() AuditEventRecordedData {
	var result AuditEventRecordedData
	if a.Event != nil {
		var tmp AuditEvent
		tmp = (*a.Event).Copy() // model
		result.Event = &tmp
	}

	return result
}
func (a *AuditEventRecordedData) Equals(to *AuditEventRecordedData) bool {
	if (a == nil) != (to == nil) {
		return false
	}
	if a == nil && to == nil {
		return true
	}
	if (a.Event == nil) != (to.Event == nil) {
		return false
	}
	if a.Event != nil && to.Event != nil {
		if !(*a.Event).Equals(&(*to.Event)) {
			return false
		}
	}

	return true
}
