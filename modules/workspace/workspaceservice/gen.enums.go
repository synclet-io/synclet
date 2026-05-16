package workspaceservice

import (
	strconv "strconv"
	// user code 'imports'
	// end user code 'imports'
)

type MemberRole byte

const (
	MemberRoleAdmin MemberRole = iota + 1
	MemberRoleEditor
	MemberRoleViewer
)

// user code 'MemberRole methods'
// end user code 'MemberRole methods'
func (m MemberRole) IsValid() bool {
	return m > 0 && m < 4
}
func (m MemberRole) IsAdmin() bool {
	return m == MemberRoleAdmin
}
func (m MemberRole) IsEditor() bool {
	return m == MemberRoleEditor
}
func (m MemberRole) IsViewer() bool {
	return m == MemberRoleViewer
}
func (m MemberRole) String() string {
	const names = "AdminEditorViewer"

	var indexes = [...]int32{0, 5, 11, 17}
	if m < 1 || m > 3 {
		return "MemberRole(" + strconv.FormatInt(int64(m), 10) + ")"
	}

	return names[indexes[m-1]:indexes[m]]
}

type InviteStatus byte

const (
	InviteStatusPending InviteStatus = iota + 1
	InviteStatusAccepted
	InviteStatusDeclined
	InviteStatusRevoked
)

// user code 'InviteStatus methods'
// end user code 'InviteStatus methods'
func (i InviteStatus) IsValid() bool {
	return i > 0 && i < 5
}
func (i InviteStatus) IsPending() bool {
	return i == InviteStatusPending
}
func (i InviteStatus) IsAccepted() bool {
	return i == InviteStatusAccepted
}
func (i InviteStatus) IsDeclined() bool {
	return i == InviteStatusDeclined
}
func (i InviteStatus) IsRevoked() bool {
	return i == InviteStatusRevoked
}
func (i InviteStatus) String() string {
	const names = "PendingAcceptedDeclinedRevoked"

	var indexes = [...]int32{0, 7, 15, 23, 30}
	if i < 1 || i > 4 {
		return "InviteStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}

	return names[indexes[i-1]:indexes[i]]
}

type WorkspaceEventType byte

const (
	WorkspaceEventTypeCreated WorkspaceEventType = iota + 1
	WorkspaceEventTypeUpdated
	WorkspaceEventTypeDeleted
)

// user code 'WorkspaceEventType methods'
// end user code 'WorkspaceEventType methods'
func (w WorkspaceEventType) IsValid() bool {
	return w > 0 && w < 4
}
func (w WorkspaceEventType) IsCreated() bool {
	return w == WorkspaceEventTypeCreated
}
func (w WorkspaceEventType) IsUpdated() bool {
	return w == WorkspaceEventTypeUpdated
}
func (w WorkspaceEventType) IsDeleted() bool {
	return w == WorkspaceEventTypeDeleted
}
