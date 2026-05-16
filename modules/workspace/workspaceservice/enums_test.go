package workspaceservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemberRole_IsValid(t *testing.T) {
	tests := []struct {
		role MemberRole
		want bool
	}{
		{MemberRole(0), false},
		{MemberRoleAdmin, true},
		{MemberRoleEditor, true},
		{MemberRoleViewer, true},
		{MemberRole(4), false},
		{MemberRole(255), false},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, tc.role.IsValid(), "role=%v", tc.role)
	}
}

func TestMemberRole_Predicates(t *testing.T) {
	assert.True(t, MemberRoleAdmin.IsAdmin())
	assert.False(t, MemberRoleEditor.IsAdmin())
	assert.False(t, MemberRoleViewer.IsAdmin())

	assert.True(t, MemberRoleEditor.IsEditor())
	assert.False(t, MemberRoleAdmin.IsEditor())
	assert.False(t, MemberRoleViewer.IsEditor())

	assert.True(t, MemberRoleViewer.IsViewer())
	assert.False(t, MemberRoleAdmin.IsViewer())
	assert.False(t, MemberRoleEditor.IsViewer())
}

func TestMemberRole_String(t *testing.T) {
	tests := []struct {
		role MemberRole
		want string
	}{
		{MemberRoleAdmin, "Admin"},
		{MemberRoleEditor, "Editor"},
		{MemberRoleViewer, "Viewer"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, tc.role.String())
	}

	// Out-of-range values fall back to a debug representation rather than panicking
	// or returning garbage — important because Role values come over the wire.
	assert.Contains(t, MemberRole(0).String(), "MemberRole(")
	assert.Contains(t, MemberRole(99).String(), "MemberRole(")
}

func TestInviteStatus_IsValid(t *testing.T) {
	tests := []struct {
		status InviteStatus
		want   bool
	}{
		{InviteStatus(0), false},
		{InviteStatusPending, true},
		{InviteStatusAccepted, true},
		{InviteStatusDeclined, true},
		{InviteStatusRevoked, true},
		{InviteStatus(5), false},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, tc.status.IsValid(), "status=%v", tc.status)
	}
}

func TestInviteStatus_Predicates(t *testing.T) {
	assert.True(t, InviteStatusPending.IsPending())
	assert.True(t, InviteStatusAccepted.IsAccepted())
	assert.True(t, InviteStatusDeclined.IsDeclined())
	assert.True(t, InviteStatusRevoked.IsRevoked())

	assert.False(t, InviteStatusPending.IsAccepted())
	assert.False(t, InviteStatusAccepted.IsPending())
}

func TestInviteStatus_String(t *testing.T) {
	tests := []struct {
		status InviteStatus
		want   string
	}{
		{InviteStatusPending, "Pending"},
		{InviteStatusAccepted, "Accepted"},
		{InviteStatusDeclined, "Declined"},
		{InviteStatusRevoked, "Revoked"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, tc.status.String())
	}

	assert.Contains(t, InviteStatus(0).String(), "InviteStatus(")
}
