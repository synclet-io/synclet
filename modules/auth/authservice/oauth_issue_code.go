package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// IssueOAuthCodeParams is what a user approved on the consent page.
type IssueOAuthCodeParams struct {
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              uuid.UUID
	WorkspaceID         uuid.UUID
	Level               OAuthAccessLevel
}

// IssueOAuthCode issues a one-time authorization code.
type IssueOAuthCode struct {
	storage Storage
	members WorkspaceMembershipLister
}

// NewIssueOAuthCode creates a new IssueOAuthCode use case.
func NewIssueOAuthCode(storage Storage, members WorkspaceMembershipLister) *IssueOAuthCode {
	return &IssueOAuthCode{storage: storage, members: members}
}

// Execute stores a code bound to the client, redirect URI, PKCE challenge,
// user, workspace and level, valid for OAuthCodeTTL, and returns it raw. A
// level above what the user's role in the workspace allows is refused.
func (uc *IssueOAuthCode) Execute(ctx context.Context, params IssueOAuthCodeParams) (string, error) {
	if params.CodeChallengeMethod != oauthChallengeMethodS256 {
		return "", &ValidationError{Message: "code_challenge_method must be S256"}
	}

	if params.CodeChallenge == "" {
		return "", &ValidationError{Message: "code_challenge is required"}
	}

	if !params.Level.IsValid() {
		return "", &ValidationError{Message: "access level is required"}
	}

	if _, err := getOAuthClient(ctx, uc.storage, params.ClientID, params.RedirectURI); err != nil {
		return "", err
	}

	role, err := workspaceRole(ctx, uc.members, params.UserID, params.WorkspaceID)
	if err != nil {
		return "", err
	}

	if role == WorkspaceRoleUnspecified {
		return "", &ValidationError{Message: "user is not a member of the workspace"}
	}

	if role == WorkspaceRoleViewer && params.Level != OAuthAccessLevelRead {
		return "", &ValidationError{Message: "a viewer can approve read access only"}
	}

	code, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("generating authorization code: %w", err)
	}

	now := time.Now()
	if _, err := uc.storage.OAuthCodes().Create(ctx, &OAuthCode{
		ID:            uuid.New(),
		CodeHash:      hashToken(code),
		ClientID:      params.ClientID,
		RedirectURI:   params.RedirectURI,
		CodeChallenge: params.CodeChallenge,
		UserID:        params.UserID,
		WorkspaceID:   params.WorkspaceID,
		Level:         params.Level,
		ExpiresAt:     now.Add(OAuthCodeTTL),
		CreatedAt:     now,
	}); err != nil {
		return "", fmt.Errorf("creating authorization code: %w", err)
	}

	return code, nil
}
