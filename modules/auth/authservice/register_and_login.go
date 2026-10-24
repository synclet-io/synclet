package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// WorkspaceAutoAssigner assigns a newly registered user to a default workspace.
type WorkspaceAutoAssigner interface {
	AutoAssign(ctx context.Context, userID uuid.UUID) error
}

// RegisterAndLoginParams holds parameters for registering and logging in.
type RegisterAndLoginParams struct {
	Email    string
	Password string
	Name     string
}

// RegisterAndLoginResult holds the result of register-and-login.
type RegisterAndLoginResult struct {
	Tokens     *TokenPair
	User       *User
	Workspaces []WorkspaceMembership
}

// RegisterAndLogin combines user registration with automatic login.
type RegisterAndLogin struct {
	registrationOpen      *RegistrationOpen
	register              *Register
	login                 *Login
	workspaceAutoAssigner WorkspaceAutoAssigner
	listUserWorkspaces    *ListUserWorkspaces
	audit                 AuditRecorder
	config                Config
}

// NewRegisterAndLogin creates a new RegisterAndLogin use case.
func NewRegisterAndLogin(registrationOpen *RegistrationOpen, register *Register, login *Login, workspaceAutoAssigner WorkspaceAutoAssigner, listUserWorkspaces *ListUserWorkspaces, audit AuditRecorder, config Config) *RegisterAndLogin {
	return &RegisterAndLogin{
		registrationOpen:      registrationOpen,
		register:              register,
		login:                 login,
		workspaceAutoAssigner: workspaceAutoAssigner,
		listUserWorkspaces:    listUserWorkspaces,
		audit:                 audit,
		config:                config,
	}
}

// Execute registers a new user and immediately logs them in.
func (uc *RegisterAndLogin) Execute(ctx context.Context, params RegisterAndLoginParams) (*RegisterAndLoginResult, error) {
	open, err := uc.registrationOpen.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking registration: %w", err)
	}

	if !open {
		return nil, ErrRegistrationDisabled
	}

	user, err := uc.register.Execute(ctx, params.Email, params.Password, params.Name)
	if err != nil {
		return nil, fmt.Errorf("registering user: %w", err)
	}

	if uc.config.SingleWorkspaceMode {
		if err := uc.workspaceAutoAssigner.AutoAssign(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("auto-assigning to default workspace: %w", err)
		}
	}

	recordUserCreated(ctx, uc.audit, user)

	tokens, err := uc.login.Execute(ctx, params.Email, params.Password)
	if err != nil {
		return nil, fmt.Errorf("auto-login after registration: %w", err)
	}

	workspaces, err := uc.listUserWorkspaces.Execute(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("listing user workspaces: %w", err)
	}

	return &RegisterAndLoginResult{
		Tokens:     tokens,
		User:       user,
		Workspaces: workspaces,
	}, nil
}

func recordUserCreated(ctx context.Context, audit AuditRecorder, user *User) {
	audit.Record(ctx, AuditEvent{
		ActorID:       user.ID,
		ActorLabel:    user.Email,
		Action:        "create",
		ResourceType:  "user",
		ResourceID:    user.ID,
		ResourceLabel: user.Email,
		After:         map[string]any{"email": user.Email, "name": user.Name},
	})
}
