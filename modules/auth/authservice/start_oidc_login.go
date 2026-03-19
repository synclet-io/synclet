package authservice

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// StartOIDCLogin generates the authorization URL for the given provider.
type StartOIDCLogin struct {
	providers  map[string]*OIDCProvider
	stateStore *StateStore
}

// NewStartOIDCLogin creates a new StartOIDCLogin use case.
func NewStartOIDCLogin(providers map[string]*OIDCProvider, stateStore *StateStore) *StartOIDCLogin {
	return &StartOIDCLogin{providers: providers, stateStore: stateStore}
}

// Execute generates the authorization URL for the given provider slug.
// Returns the full auth URL to redirect the user to.
func (uc *StartOIDCLogin) Execute(ctx context.Context, providerSlug string) (authURL string, err error) {
	p, ok := uc.providers[providerSlug]
	if !ok {
		return "", fmt.Errorf("unknown OIDC provider: %s", providerSlug)
	}
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	verifier := oauth2.GenerateVerifier()
	if err := uc.stateStore.Set(ctx, state, verifier, providerSlug, 10*time.Minute); err != nil {
		return "", fmt.Errorf("storing state: %w", err)
	}
	authURL = p.oauth2Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
	return authURL, nil
}
