package workspacestorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	time "time"

	uuid "github.com/google/uuid"

	workspaceservice "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonWorkspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (w *jsonWorkspace) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspace) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceToJsonModel(src *workspaceservice.Workspace) (*jsonWorkspace, error) {
	result := &jsonWorkspace{}
	result.ID = src.ID
	result.Name = src.Name
	result.Slug = src.Slug
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertWorkspaceFromJsonModel(src *jsonWorkspace) (*workspaceservice.Workspace, error) {
	result := &workspaceservice.Workspace{}
	result.ID = src.ID
	result.Name = src.Name
	result.Slug = src.Slug
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonWorkspaceMember struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

func (w *jsonWorkspaceMember) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceMember) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceMemberToJsonModel(src *workspaceservice.WorkspaceMember) (*jsonWorkspaceMember, error) {
	result := &jsonWorkspaceMember{}
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

func convertWorkspaceMemberFromJsonModel(src *jsonWorkspaceMember) (*workspaceservice.WorkspaceMember, error) {
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

type jsonWorkspaceInvite struct {
	ID            uuid.UUID `json:"id"`
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	InviterUserID uuid.UUID `json:"inviter_user_id"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	Token         string    `json:"token"`
	Status        string    `json:"status"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (w *jsonWorkspaceInvite) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceInvite) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceInviteToJsonModel(src *workspaceservice.WorkspaceInvite) (*jsonWorkspaceInvite, error) {
	result := &jsonWorkspaceInvite{}
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

func convertWorkspaceInviteFromJsonModel(src *jsonWorkspaceInvite) (*workspaceservice.WorkspaceInvite, error) {
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
