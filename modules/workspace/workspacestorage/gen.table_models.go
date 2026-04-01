package workspacestorage

import (
	time "time"

	uuid "github.com/google/uuid"

	workspaceservice "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbWorkspace struct {
	ID        uuid.UUID `gorm:"column:id;"`
	Name      string    `gorm:"column:name;type:text;"`
	Slug      string    `gorm:"column:slug;type:text;"`
	CreatedAt time.Time `gorm:"column:created_at;"`
	UpdatedAt time.Time `gorm:"column:updated_at;"`
}

func convertWorkspaceToDB(src *workspaceservice.Workspace) (*dbWorkspace, error) {
	result := &dbWorkspace{}
	result.ID = src.ID
	result.Name = src.Name
	result.Slug = src.Slug
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()

	return result, nil
}

func convertWorkspaceFromDB(src *dbWorkspace) (*workspaceservice.Workspace, error) {
	result := &workspaceservice.Workspace{}
	result.ID = src.ID
	result.Name = src.Name
	result.Slug = src.Slug
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt

	return result, nil
}
func (a dbWorkspace) TableName() string {
	return "workspace.workspaces"
}

type dbWorkspaceMember struct {
	ID          uuid.UUID `gorm:"column:id;"`
	WorkspaceID uuid.UUID `gorm:"column:workspace_id;"`
	UserID      uuid.UUID `gorm:"column:user_id;"`
	Role        string    `gorm:"column:role;type:text;"`
	JoinedAt    time.Time `gorm:"column:joined_at;"`
}

func convertWorkspaceMemberToDB(src *workspaceservice.WorkspaceMember) (*dbWorkspaceMember, error) {
	result := &dbWorkspaceMember{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.UserID = src.UserID
	tmp3, err := convertMemberRoleToDB(src.Role)
	if err != nil {
		return nil, err
	}
	result.Role = tmp3
	result.JoinedAt = (src.JoinedAt).UTC()

	return result, nil
}

func convertWorkspaceMemberFromDB(src *dbWorkspaceMember) (*workspaceservice.WorkspaceMember, error) {
	result := &workspaceservice.WorkspaceMember{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.UserID = src.UserID
	tmp8, err := convertMemberRoleFromDB(src.Role)
	if err != nil {
		return nil, err
	}
	result.Role = tmp8
	result.JoinedAt = src.JoinedAt

	return result, nil
}
func (a dbWorkspaceMember) TableName() string {
	return "workspace.workspace_members"
}

type dbWorkspaceInvite struct {
	ID            uuid.UUID `gorm:"column:id;"`
	WorkspaceID   uuid.UUID `gorm:"column:workspace_id;"`
	InviterUserID uuid.UUID `gorm:"column:inviter_user_id;"`
	Email         string    `gorm:"column:email;type:text;"`
	Role          string    `gorm:"column:role;type:text;"`
	Token         string    `gorm:"column:token;type:text;"`
	Status        string    `gorm:"column:status;type:text;"`
	ExpiresAt     time.Time `gorm:"column:expires_at;"`
	CreatedAt     time.Time `gorm:"column:created_at;"`
	UpdatedAt     time.Time `gorm:"column:updated_at;"`
}

func convertWorkspaceInviteToDB(src *workspaceservice.WorkspaceInvite) (*dbWorkspaceInvite, error) {
	result := &dbWorkspaceInvite{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.InviterUserID = src.InviterUserID
	result.Email = src.Email
	tmp4, err := convertMemberRoleToDB(src.Role)
	if err != nil {
		return nil, err
	}
	result.Role = tmp4
	result.Token = src.Token
	tmp6, err := convertInviteStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp6
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()

	return result, nil
}

func convertWorkspaceInviteFromDB(src *dbWorkspaceInvite) (*workspaceservice.WorkspaceInvite, error) {
	result := &workspaceservice.WorkspaceInvite{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.InviterUserID = src.InviterUserID
	result.Email = src.Email
	tmp14, err := convertMemberRoleFromDB(src.Role)
	if err != nil {
		return nil, err
	}
	result.Role = tmp14
	result.Token = src.Token
	tmp16, err := convertInviteStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp16
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt

	return result, nil
}
func (a dbWorkspaceInvite) TableName() string {
	return "workspace.workspace_invites"
}
