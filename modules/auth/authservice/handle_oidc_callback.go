package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"golang.org/x/oauth2"
)

// HandleOIDCCallback processes the OIDC callback, exchanges the code, verifies
// the ID token, validates claims, and returns Synclet JWT tokens.
type HandleOIDCCallback struct {
	providers             map[string]*OIDCProvider
	stateStore            *StateStore
	storage               Storage
	config                Config
	workspaceAutoAssigner WorkspaceAutoAssigner
	singleWorkspaceMode   bool
	logger                *logging.Logger
}

// NewHandleOIDCCallback creates a new HandleOIDCCallback use case.
func NewHandleOIDCCallback(providers map[string]*OIDCProvider, stateStore *StateStore, storage Storage, config Config, workspaceAutoAssigner WorkspaceAutoAssigner, singleWorkspaceMode bool, logger *logging.Logger) *HandleOIDCCallback {
	return &HandleOIDCCallback{
		providers:             providers,
		stateStore:            stateStore,
		storage:               storage,
		config:                config,
		workspaceAutoAssigner: workspaceAutoAssigner,
		singleWorkspaceMode:   singleWorkspaceMode,
		logger:                logger,
	}
}

// Execute processes the OIDC callback for the given provider, code, and state.
func (uc *HandleOIDCCallback) Execute(ctx context.Context, providerSlug, code, state string) (*TokenPair, error) {
	verifier, storedProvider, ok := uc.stateStore.Get(ctx, state)
	if !ok {
		return nil, errors.New("invalid or expired state")
	}

	if storedProvider != providerSlug {
		return nil, errors.New("state provider mismatch")
	}

	provider, ok := uc.providers[providerSlug]
	if !ok {
		return nil, fmt.Errorf("unknown OIDC provider: %s", providerSlug)
	}

	// Exchange authorization code for tokens with PKCE verifier.
	oauth2Token, err := provider.oauth2Config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	// Extract and verify ID token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("missing id_token in token response")
	}

	idToken, err := provider.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verifying id_token: %w", err)
	}

	// Extract all claims as raw map for bound claims and role mapping.
	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err != nil {
		return nil, fmt.Errorf("extracting claims: %w", err)
	}

	// Extract standard claims.
	email, _ := allClaims["email"].(string)
	emailVerified, _ := allClaims["email_verified"].(bool)
	name, _ := allClaims["name"].(string)
	sub := idToken.Subject

	// Validate bound claims.
	if err := ValidateBoundClaims(provider.Config.BoundClaims, allClaims); err != nil {
		return nil, fmt.Errorf("bound claims validation: %w", err)
	}

	// Validate email domain.
	if err := ValidateEmailDomain(email, emailVerified, provider.Config.AllowedDomains); err != nil {
		return nil, fmt.Errorf("email domain validation: %w", err)
	}

	// Map role from claims.
	role := MapRole(provider.Config.RoleClaim, provider.Config.RoleMapping, provider.Config.DefaultRole, allClaims)
	_ = role // Role will be used for workspace membership in future; stored for now.

	// Find or create user and OIDC identity link.
	user, err := uc.findOrCreateUser(ctx, provider.Config, sub, email, name)
	if err != nil {
		return nil, fmt.Errorf("find or create user: %w", err)
	}

	// Issue Synclet JWT tokens.
	return generateTokenPair(ctx, uc.storage, uc.config, user)
}

// findOrCreateUser links an OIDC identity to a Synclet user.
// 1. Check oidc_identities for existing link (provider_slug + subject)
// 2. If not found, look up user by email
// 3. If no user, auto-create (if enabled)
// 4. Create oidc_identity link
func (uc *HandleOIDCCallback) findOrCreateUser(ctx context.Context, cfg OIDCProviderConfig, subject, email, name string) (*User, error) {
	// Check for existing OIDC identity link.
	existing, err := uc.storage.OIDCIdentitys().First(ctx, &OIDCIdentityFilter{
		ProviderSlug: filter.Equals(cfg.Slug),
		Subject:      filter.Equals(subject),
	})
	if err == nil {
		// Update last login time.
		existing.LastLoginAt = time.Now()
		if _, err := uc.storage.OIDCIdentitys().Update(ctx, existing); err != nil {
			uc.logger.WithError(err).WithField("identity_id", existing.ID).Warn(ctx, "failed to update OIDC identity last login time")
		}
		// Return the linked user.
		user, err := uc.storage.Users().First(ctx, &UserFilter{ID: filter.Equals(existing.UserID)})
		if err != nil {
			return nil, fmt.Errorf("loading linked user: %w", err)
		}

		return user, nil
	}

	// No existing link. Find user by email.
	var user *User
	if email != "" {
		user, err = uc.storage.Users().First(ctx, &UserFilter{Email: filter.Equals(email)})
		if err != nil {
			user = nil // Not found, will auto-create.
		}
	}

	// Auto-create user if needed.
	if user == nil {
		if !cfg.AutoCreateUser {
			return nil, fmt.Errorf("user not found and auto-creation disabled for provider %s", cfg.Slug)
		}

		if name == "" {
			name = email
		}

		user = &User{
			ID:           uuid.New(),
			Email:        email,
			PasswordHash: "", // OIDC-only user, no password.
			Name:         name,
			CreatedAt:    time.Now(),
		}

		user, err = uc.storage.Users().Create(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("creating user: %w", err)
		}

		// Auto-assign to default workspace in single-workspace mode.
		if uc.singleWorkspaceMode {
			if err := uc.workspaceAutoAssigner.AutoAssign(ctx, user.ID); err != nil {
				return nil, fmt.Errorf("auto-assigning to default workspace: %w", err)
			}
		}
	}

	// Create OIDC identity link.
	identity := &OIDCIdentity{
		ID:           uuid.New(),
		UserID:       user.ID,
		ProviderSlug: cfg.Slug,
		Subject:      subject,
		Email:        email,
		CreatedAt:    time.Now(),
		LastLoginAt:  time.Now(),
	}
	if _, err := uc.storage.OIDCIdentitys().Create(ctx, identity); err != nil {
		return nil, fmt.Errorf("creating oidc identity: %w", err)
	}

	return user, nil
}
