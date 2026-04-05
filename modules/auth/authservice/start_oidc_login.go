package authservice

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// StartOIDCLogin generates the authorization URL for the given provider.
type StartOIDCLogin struct {
	providers  map[string]*OIDCProvider
	stateStore *StateStore
	cfg        Config
}

// NewStartOIDCLogin creates a new StartOIDCLogin use case.
func NewStartOIDCLogin(providers map[string]*OIDCProvider, stateStore *StateStore, cfg Config) *StartOIDCLogin {
	return &StartOIDCLogin{providers: providers, stateStore: stateStore, cfg: cfg}
}

// Execute generates the authorization URL for the given provider slug.
// Returns the full auth URL to redirect the user to.
func (uc *StartOIDCLogin) Execute(ctx context.Context, providerSlug string) (authURL string, err error) {
	provider, ok := uc.providers[providerSlug]
	if !ok {
		return "", fmt.Errorf("unknown OIDC provider: %s", providerSlug)
	}

	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}

	verifier := oauth2.GenerateVerifier()
	if err := uc.stateStore.Set(ctx, state, verifier, providerSlug, uc.cfg.OIDCStateTTL); err != nil {
		return "", fmt.Errorf("storing state: %w", err)
	}

	authURL = provider.oauth2Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	return authURL, nil
}
