package workspaceservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

type WorkspaceEventData interface {
	isWorkspaceEventData()
	WorkspaceEventDataEquals(WorkspaceEventData) bool
	// user code 'WorkspaceEventData methods'
	// end user code 'WorkspaceEventData methods'
}

func (*WorkspaceCreatedEventData) isWorkspaceEventData() {}
func (w *WorkspaceCreatedEventData) WorkspaceEventDataEquals(to WorkspaceEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*WorkspaceCreatedEventData)
	if !ok {
		return false
	}

	return w.Equals(toTyped)
}
func (*WorkspaceUpdatedEventData) isWorkspaceEventData() {}
func (w *WorkspaceUpdatedEventData) WorkspaceEventDataEquals(to WorkspaceEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*WorkspaceUpdatedEventData)
	if !ok {
		return false
	}

	return w.Equals(toTyped)
}
func (*WorkspaceDeletedEventData) isWorkspaceEventData() {}
func (w *WorkspaceDeletedEventData) WorkspaceEventDataEquals(to WorkspaceEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*WorkspaceDeletedEventData)
	if !ok {
		return false
	}

	return w.Equals(toTyped)
}

func copyWorkspaceEventData(val WorkspaceEventData) WorkspaceEventData {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case *WorkspaceCreatedEventData:
		valCopy := val.Copy()
		return &valCopy
	case *WorkspaceUpdatedEventData:
		valCopy := val.Copy()
		return &valCopy
	case *WorkspaceDeletedEventData:
		valCopy := val.Copy()
		return &valCopy
	}
	panic("called copyWorkspaceEventData with invalid type")
}

type WorkspaceField byte

const (
	WorkspaceFieldID WorkspaceField = iota + 1
	WorkspaceFieldName
	WorkspaceFieldSlug
	WorkspaceFieldCreatedAt
	WorkspaceFieldUpdatedAt
)

type WorkspaceFilter struct {
	ID   filter.Filter[uuid.UUID]
	Name filter.Filter[string]
	Slug filter.Filter[string]
	Or   []*WorkspaceFilter
	And  []*WorkspaceFilter
}
type WorkspaceOrder order.Order[WorkspaceField]

type Workspace struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// user code 'Workspace methods'
// end user code 'Workspace methods'

func (w *Workspace) Copy() Workspace {
	var result Workspace
	result.ID = w.ID
	result.Name = w.Name
	result.Slug = w.Slug
	result.CreatedAt = w.CreatedAt
	result.UpdatedAt = w.UpdatedAt

	return result
}
func (w *Workspace) Equals(to *Workspace) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.Name != to.Name {
		return false
	}
	if w.Slug != to.Slug {
		return false
	}
	if w.CreatedAt != to.CreatedAt {
		return false
	}
	if w.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type WorkspaceMemberField byte

const (
	WorkspaceMemberFieldID WorkspaceMemberField = iota + 1
	WorkspaceMemberFieldWorkspaceID
	WorkspaceMemberFieldUserID
	WorkspaceMemberFieldRole
	WorkspaceMemberFieldJoinedAt
)

type WorkspaceMemberFilter struct {
	ID          filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	UserID      filter.Filter[uuid.UUID]
	Role        filter.Filter[MemberRole]
	Or          []*WorkspaceMemberFilter
	And         []*WorkspaceMemberFilter
}
type WorkspaceMemberOrder order.Order[WorkspaceMemberField]

type WorkspaceMember struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        MemberRole
	JoinedAt    time.Time
}

// user code 'WorkspaceMember methods'
// end user code 'WorkspaceMember methods'

func (w *WorkspaceMember) Copy() WorkspaceMember {
	var result WorkspaceMember
	result.ID = w.ID
	result.WorkspaceID = w.WorkspaceID
	result.UserID = w.UserID
	result.Role = w.Role // enum
	result.JoinedAt = w.JoinedAt

	return result
}
func (w *WorkspaceMember) Equals(to *WorkspaceMember) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.UserID != to.UserID {
		return false
	}
	if w.Role != to.Role {
		return false
	}
	if w.JoinedAt != to.JoinedAt {
		return false
	}

	return true
}

type WorkspaceInviteField byte

const (
	WorkspaceInviteFieldID WorkspaceInviteField = iota + 1
	WorkspaceInviteFieldWorkspaceID
	WorkspaceInviteFieldInviterUserID
	WorkspaceInviteFieldEmail
	WorkspaceInviteFieldRole
	WorkspaceInviteFieldToken
	WorkspaceInviteFieldStatus
	WorkspaceInviteFieldExpiresAt
	WorkspaceInviteFieldCreatedAt
	WorkspaceInviteFieldUpdatedAt
)

type WorkspaceInviteFilter struct {
	ID            filter.Filter[uuid.UUID]
	WorkspaceID   filter.Filter[uuid.UUID]
	InviterUserID filter.Filter[uuid.UUID]
	Email         filter.Filter[string]
	Token         filter.Filter[string]
	Status        filter.Filter[InviteStatus]
	Or            []*WorkspaceInviteFilter
	And           []*WorkspaceInviteFilter
}
type WorkspaceInviteOrder order.Order[WorkspaceInviteField]

type WorkspaceInvite struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	InviterUserID uuid.UUID
	Email         string
	Role          MemberRole
	Token         string
	Status        InviteStatus
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// user code 'WorkspaceInvite methods'
// end user code 'WorkspaceInvite methods'

func (w *WorkspaceInvite) Copy() WorkspaceInvite {
	var result WorkspaceInvite
	result.ID = w.ID
	result.WorkspaceID = w.WorkspaceID
	result.InviterUserID = w.InviterUserID
	result.Email = w.Email
	result.Role = w.Role // enum
	result.Token = w.Token
	result.Status = w.Status // enum
	result.ExpiresAt = w.ExpiresAt
	result.CreatedAt = w.CreatedAt
	result.UpdatedAt = w.UpdatedAt

	return result
}
func (w *WorkspaceInvite) Equals(to *WorkspaceInvite) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.InviterUserID != to.InviterUserID {
		return false
	}
	if w.Email != to.Email {
		return false
	}
	if w.Role != to.Role {
		return false
	}
	if w.Token != to.Token {
		return false
	}
	if w.Status != to.Status {
		return false
	}
	if w.ExpiresAt != to.ExpiresAt {
		return false
	}
	if w.CreatedAt != to.CreatedAt {
		return false
	}
	if w.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type WorkspaceEventField byte

const (
	WorkspaceEventFieldEventType WorkspaceEventField = iota + 1
	WorkspaceEventFieldData
)

type WorkspaceEventFilter struct {
	Or  []*WorkspaceEventFilter
	And []*WorkspaceEventFilter
}
type WorkspaceEventOrder order.Order[WorkspaceEventField]

type WorkspaceEvent struct {
	EventType WorkspaceEventType
	Data      WorkspaceEventData
}

// user code 'WorkspaceEvent methods'
// end user code 'WorkspaceEvent methods'

func (w *WorkspaceEvent) Copy() WorkspaceEvent {
	var result WorkspaceEvent
	result.EventType = w.EventType // enum
	result.Data = copyWorkspaceEventData(w.Data)

	return result
}
func (w *WorkspaceEvent) Equals(to *WorkspaceEvent) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.EventType != to.EventType {
		return false
	}
	if !w.Data.WorkspaceEventDataEquals(to.Data) {
		return false
	}

	return true
}

type WorkspaceCreatedEventDataField byte

const (
	WorkspaceCreatedEventDataFieldWorkspace WorkspaceCreatedEventDataField = iota + 1
)

type WorkspaceCreatedEventDataFilter struct {
	Or  []*WorkspaceCreatedEventDataFilter
	And []*WorkspaceCreatedEventDataFilter
}
type WorkspaceCreatedEventDataOrder order.Order[WorkspaceCreatedEventDataField]

type WorkspaceCreatedEventData struct {
	Workspace *Workspace
}

// user code 'WorkspaceCreatedEventData methods'
// end user code 'WorkspaceCreatedEventData methods'

func (w *WorkspaceCreatedEventData) Copy() WorkspaceCreatedEventData {
	var result WorkspaceCreatedEventData
	if w.Workspace != nil {
		var tmp Workspace
		tmp = (*w.Workspace).Copy() // model
		result.Workspace = &tmp
	}

	return result
}
func (w *WorkspaceCreatedEventData) Equals(to *WorkspaceCreatedEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if (w.Workspace == nil) != (to.Workspace == nil) {
		return false
	}
	if w.Workspace != nil && to.Workspace != nil {
		if !(*w.Workspace).Equals(&(*to.Workspace)) {
			return false
		}
	}

	return true
}

type WorkspaceUpdatedEventDataField byte

const (
	WorkspaceUpdatedEventDataFieldOldData WorkspaceUpdatedEventDataField = iota + 1
	WorkspaceUpdatedEventDataFieldNewData
)

type WorkspaceUpdatedEventDataFilter struct {
	Or  []*WorkspaceUpdatedEventDataFilter
	And []*WorkspaceUpdatedEventDataFilter
}
type WorkspaceUpdatedEventDataOrder order.Order[WorkspaceUpdatedEventDataField]

type WorkspaceUpdatedEventData struct {
	OldData *Workspace
	NewData *Workspace
}

// user code 'WorkspaceUpdatedEventData methods'
// end user code 'WorkspaceUpdatedEventData methods'

func (w *WorkspaceUpdatedEventData) Copy() WorkspaceUpdatedEventData {
	var result WorkspaceUpdatedEventData
	if w.OldData != nil {
		var tmp Workspace
		tmp = (*w.OldData).Copy() // model
		result.OldData = &tmp
	}
	if w.NewData != nil {
		var tmp1 Workspace
		tmp1 = (*w.NewData).Copy() // model
		result.NewData = &tmp1
	}

	return result
}
func (w *WorkspaceUpdatedEventData) Equals(to *WorkspaceUpdatedEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if (w.OldData == nil) != (to.OldData == nil) {
		return false
	}
	if w.OldData != nil && to.OldData != nil {
		if !(*w.OldData).Equals(&(*to.OldData)) {
			return false
		}
	}
	if (w.NewData == nil) != (to.NewData == nil) {
		return false
	}
	if w.NewData != nil && to.NewData != nil {
		if !(*w.NewData).Equals(&(*to.NewData)) {
			return false
		}
	}

	return true
}

type WorkspaceDeletedEventDataField byte

const (
	WorkspaceDeletedEventDataFieldWorkspace WorkspaceDeletedEventDataField = iota + 1
)

type WorkspaceDeletedEventDataFilter struct {
	Or  []*WorkspaceDeletedEventDataFilter
	And []*WorkspaceDeletedEventDataFilter
}
type WorkspaceDeletedEventDataOrder order.Order[WorkspaceDeletedEventDataField]

type WorkspaceDeletedEventData struct {
	Workspace *Workspace
}

// user code 'WorkspaceDeletedEventData methods'
// end user code 'WorkspaceDeletedEventData methods'

func (w *WorkspaceDeletedEventData) Copy() WorkspaceDeletedEventData {
	var result WorkspaceDeletedEventData
	if w.Workspace != nil {
		var tmp Workspace
		tmp = (*w.Workspace).Copy() // model
		result.Workspace = &tmp
	}

	return result
}
func (w *WorkspaceDeletedEventData) Equals(to *WorkspaceDeletedEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if (w.Workspace == nil) != (to.Workspace == nil) {
		return false
	}
	if w.Workspace != nil && to.Workspace != nil {
		if !(*w.Workspace).Equals(&(*to.Workspace)) {
			return false
		}
	}

	return true
}
