package workspacestorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	fmt "fmt"
	time "time"

	uuid "github.com/google/uuid"
	errors "github.com/pkg/errors"

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

type jsonWorkspaceEvent struct {
	EventType string                  `json:"event_type"`
	Data      *jsonWorkspaceEventData `json:"data"`
}

func (w *jsonWorkspaceEvent) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceEvent) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceEventToJsonModel(src *workspaceservice.WorkspaceEvent) (*jsonWorkspaceEvent, error) {
	result := &jsonWorkspaceEvent{}
	tmp, err := convertWorkspaceEventTypeToDB(src.EventType)
	if err != nil {
		return nil, err
	}
	result.EventType = tmp
	tmp1, err := convertWorkspaceEventDataToDB(src.Data)
	if err != nil {
		return nil, err
	}
	result.Data = tmp1
	return result, nil
}

func convertWorkspaceEventFromJsonModel(src *jsonWorkspaceEvent) (*workspaceservice.WorkspaceEvent, error) {
	result := &workspaceservice.WorkspaceEvent{}
	tmp2, err := convertWorkspaceEventTypeFromDB(src.EventType)
	if err != nil {
		return nil, err
	}
	result.EventType = tmp2
	tmp3, err := convertWorkspaceEventDataFromDB(src.Data)
	if err != nil {
		return nil, fmt.Errorf("convert WorkspaceEventData to service type: %w", err)
	}
	result.Data = tmp3
	return result, nil
}

type jsonWorkspaceCreatedEventData struct {
	Workspace *jsonWorkspace `json:"workspace"`
}

func (w *jsonWorkspaceCreatedEventData) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceCreatedEventData) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceCreatedEventDataToJsonModel(src *workspaceservice.WorkspaceCreatedEventData) (*jsonWorkspaceCreatedEventData, error) {
	result := &jsonWorkspaceCreatedEventData{}
	if src.Workspace != nil {
		tmp, err := convertWorkspaceToJsonModel(src.Workspace)
		if err != nil {
			return nil, errors.Wrap(err, "convert Workspace to db")
		}
		result.Workspace = tmp
	} else {
		result.Workspace = nil
	}
	return result, nil
}

func convertWorkspaceCreatedEventDataFromJsonModel(src *jsonWorkspaceCreatedEventData) (*workspaceservice.WorkspaceCreatedEventData, error) {
	result := &workspaceservice.WorkspaceCreatedEventData{}
	if src.Workspace != nil {
		tmp1, err := convertWorkspaceFromJsonModel(src.Workspace)
		if err != nil {
			return nil, err
		}
		result.Workspace = tmp1
	} else {
		result.Workspace = nil
	}
	return result, nil
}

type jsonWorkspaceUpdatedEventData struct {
	OldData *jsonWorkspace `json:"old_data"`
	NewData *jsonWorkspace `json:"new_data"`
}

func (w *jsonWorkspaceUpdatedEventData) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceUpdatedEventData) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceUpdatedEventDataToJsonModel(src *workspaceservice.WorkspaceUpdatedEventData) (*jsonWorkspaceUpdatedEventData, error) {
	result := &jsonWorkspaceUpdatedEventData{}
	if src.OldData != nil {
		tmp, err := convertWorkspaceToJsonModel(src.OldData)
		if err != nil {
			return nil, errors.Wrap(err, "convert Workspace to db")
		}
		result.OldData = tmp
	} else {
		result.OldData = nil
	}
	if src.NewData != nil {
		tmp1, err := convertWorkspaceToJsonModel(src.NewData)
		if err != nil {
			return nil, errors.Wrap(err, "convert Workspace to db")
		}
		result.NewData = tmp1
	} else {
		result.NewData = nil
	}
	return result, nil
}

func convertWorkspaceUpdatedEventDataFromJsonModel(src *jsonWorkspaceUpdatedEventData) (*workspaceservice.WorkspaceUpdatedEventData, error) {
	result := &workspaceservice.WorkspaceUpdatedEventData{}
	if src.OldData != nil {
		tmp2, err := convertWorkspaceFromJsonModel(src.OldData)
		if err != nil {
			return nil, err
		}
		result.OldData = tmp2
	} else {
		result.OldData = nil
	}
	if src.NewData != nil {
		tmp3, err := convertWorkspaceFromJsonModel(src.NewData)
		if err != nil {
			return nil, err
		}
		result.NewData = tmp3
	} else {
		result.NewData = nil
	}
	return result, nil
}

type jsonWorkspaceDeletedEventData struct {
	Workspace *jsonWorkspace `json:"workspace"`
}

func (w *jsonWorkspaceDeletedEventData) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceDeletedEventData) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceDeletedEventDataToJsonModel(src *workspaceservice.WorkspaceDeletedEventData) (*jsonWorkspaceDeletedEventData, error) {
	result := &jsonWorkspaceDeletedEventData{}
	if src.Workspace != nil {
		tmp, err := convertWorkspaceToJsonModel(src.Workspace)
		if err != nil {
			return nil, errors.Wrap(err, "convert Workspace to db")
		}
		result.Workspace = tmp
	} else {
		result.Workspace = nil
	}
	return result, nil
}

func convertWorkspaceDeletedEventDataFromJsonModel(src *jsonWorkspaceDeletedEventData) (*workspaceservice.WorkspaceDeletedEventData, error) {
	result := &workspaceservice.WorkspaceDeletedEventData{}
	if src.Workspace != nil {
		tmp1, err := convertWorkspaceFromJsonModel(src.Workspace)
		if err != nil {
			return nil, err
		}
		result.Workspace = tmp1
	} else {
		result.Workspace = nil
	}
	return result, nil
}
