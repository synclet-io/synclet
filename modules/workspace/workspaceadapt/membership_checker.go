package workspaceadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// MembershipChecker adapts workspaceservice.GetMembership to the connectutil.MembershipChecker interface.
type MembershipChecker struct {
	getMembership *workspaceservice.GetMembership
}

// NewMembershipChecker creates a new MembershipChecker.
func NewMembershipChecker(getMembership *workspaceservice.GetMembership) *MembershipChecker {
	return &MembershipChecker{getMembership: getMembership}
}

// IsMember returns true if the user is a member of the workspace.
func (c *MembershipChecker) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	_, err := c.getMembership.Execute(ctx, workspaceID, userID)
	if err != nil {
		return false, nil //nolint:nilerr // not-found is expected, return not a member
	}

	return true, nil
}

// GetMemberRole returns the role string for the user in the workspace.
func (c *MembershipChecker) GetMemberRole(ctx context.Context, workspaceID, userID uuid.UUID) (string, error) {
	member, err := c.getMembership.Execute(ctx, workspaceID, userID)
	if err != nil {
		return "", err
	}

	return member.Role.String(), nil
}
