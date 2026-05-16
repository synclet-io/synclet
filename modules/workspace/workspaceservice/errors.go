package workspaceservice

import "errors"

// ErrLastAdminCannotBeDemoted is returned when an admin attempts to demote
// the only remaining admin of a workspace.
var ErrLastAdminCannotBeDemoted = errors.New("last admin cannot be demoted")

// ErrInvalidMemberRole is returned when the requested role is not a valid
// MemberRole value.
var ErrInvalidMemberRole = errors.New("invalid member role")
