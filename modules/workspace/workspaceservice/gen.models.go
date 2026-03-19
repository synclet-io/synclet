package workspaceservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

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
