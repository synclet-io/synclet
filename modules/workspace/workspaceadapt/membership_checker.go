package workspaceadapt

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
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
	_, err := c.GetMemberRole(ctx, workspaceID, userID)
	if errors.Is(err, connectutil.ErrNotMember) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// GetMemberRole returns the role string for the user in the workspace, or an
// error wrapping connectutil.ErrNotMember when the user is not a member.
func (c *MembershipChecker) GetMemberRole(ctx context.Context, workspaceID, userID uuid.UUID) (string, error) {
	member, err := c.getMembership.Execute(ctx, workspaceID, userID)
	if errors.Is(err, workspaceservice.ErrWorkspaceMemberNotFound) {
		return "", fmt.Errorf("%w: %w", connectutil.ErrNotMember, err)
	}

	if err != nil {
		return "", fmt.Errorf("getting membership: %w", err)
	}

	return member.Role.String(), nil
}
