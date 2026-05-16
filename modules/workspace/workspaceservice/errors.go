package workspaceservice

type baseErr string

func (e baseErr) Error() string { return string(e) }

const (
	// ErrLastAdminCannotBeDemoted is returned when an admin attempts to demote
	// the only remaining admin of a workspace.
	ErrLastAdminCannotBeDemoted baseErr = "last admin cannot be demoted"
	// ErrInvalidMemberRole is returned when the requested role is not a valid
	// MemberRole value.
	ErrInvalidMemberRole baseErr = "invalid member role"
)
