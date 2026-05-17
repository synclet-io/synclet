package auditservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

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
