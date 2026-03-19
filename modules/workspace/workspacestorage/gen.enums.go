package workspacestorage

import (
	fmt "fmt"

	workspaceservice "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	// user code 'imports'
	// end user code 'imports'
)

const (
	memberRoleAdmin  = "admin"
	memberRoleEditor = "editor"
	memberRoleViewer = "viewer"
)

func convertMemberRoleToDB(memberRoleValue workspaceservice.MemberRole) (string, error) {
	result, ok := map[workspaceservice.MemberRole]string{
		workspaceservice.MemberRoleAdmin:  memberRoleAdmin,
		workspaceservice.MemberRoleEditor: memberRoleEditor,
		workspaceservice.MemberRoleViewer: memberRoleViewer,
	}[memberRoleValue]
	if !ok {
		return "", fmt.Errorf("unknown MemberRole value: %d", memberRoleValue)
	}
	return result, nil
}

func convertMemberRoleFromDB(memberRoleValue string) (workspaceservice.MemberRole, error) {
	result, ok := map[string]workspaceservice.MemberRole{
		memberRoleAdmin:  workspaceservice.MemberRoleAdmin,
		memberRoleEditor: workspaceservice.MemberRoleEditor,
		memberRoleViewer: workspaceservice.MemberRoleViewer,
	}[memberRoleValue]
	if !ok {
		return 0, fmt.Errorf("unknown MemberRole db value: %s", memberRoleValue)
	}
	return result, nil
}

const (
	inviteStatusPending  = "pending"
	inviteStatusAccepted = "accepted"
	inviteStatusDeclined = "declined"
	inviteStatusRevoked  = "revoked"
)

func convertInviteStatusToDB(inviteStatusValue workspaceservice.InviteStatus) (string, error) {
	result, ok := map[workspaceservice.InviteStatus]string{
		workspaceservice.InviteStatusPending:  inviteStatusPending,
		workspaceservice.InviteStatusAccepted: inviteStatusAccepted,
		workspaceservice.InviteStatusDeclined: inviteStatusDeclined,
		workspaceservice.InviteStatusRevoked:  inviteStatusRevoked,
	}[inviteStatusValue]
	if !ok {
		return "", fmt.Errorf("unknown InviteStatus value: %d", inviteStatusValue)
	}
	return result, nil
}

func convertInviteStatusFromDB(inviteStatusValue string) (workspaceservice.InviteStatus, error) {
	result, ok := map[string]workspaceservice.InviteStatus{
		inviteStatusPending:  workspaceservice.InviteStatusPending,
		inviteStatusAccepted: workspaceservice.InviteStatusAccepted,
		inviteStatusDeclined: workspaceservice.InviteStatusDeclined,
		inviteStatusRevoked:  workspaceservice.InviteStatusRevoked,
	}[inviteStatusValue]
	if !ok {
		return 0, fmt.Errorf("unknown InviteStatus db value: %s", inviteStatusValue)
	}
	return result, nil
}
