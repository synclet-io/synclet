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
	Tokens *TokenPair
	User   *User
}

// RegisterAndLogin combines user registration with automatic login.
type RegisterAndLogin struct {
	register              *Register
	login                 *Login
	workspaceAutoAssigner WorkspaceAutoAssigner
	singleWorkspaceMode   bool
}

// NewRegisterAndLogin creates a new RegisterAndLogin use case.
func NewRegisterAndLogin(register *Register, login *Login, workspaceAutoAssigner WorkspaceAutoAssigner, singleWorkspaceMode bool) *RegisterAndLogin {
	return &RegisterAndLogin{
		register:              register,
		login:                 login,
		workspaceAutoAssigner: workspaceAutoAssigner,
		singleWorkspaceMode:   singleWorkspaceMode,
	}
}

// Execute registers a new user and immediately logs them in.
func (uc *RegisterAndLogin) Execute(ctx context.Context, params RegisterAndLoginParams) (*RegisterAndLoginResult, error) {
	user, err := uc.register.Execute(ctx, params.Email, params.Password, params.Name)
	if err != nil {
		return nil, fmt.Errorf("registering user: %w", err)
	}

	if uc.singleWorkspaceMode {
		if err := uc.workspaceAutoAssigner.AutoAssign(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("auto-assigning to default workspace: %w", err)
		}
	}

	tokens, err := uc.login.Execute(ctx, params.Email, params.Password)
	if err != nil {
		return nil, fmt.Errorf("auto-login after registration: %w", err)
	}

	return &RegisterAndLoginResult{
		Tokens: tokens,
		User:   user,
	}, nil
}
