package authservice

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProvider wraps a coreos/go-oidc provider with its OAuth2 config.
type OIDCProvider struct {
	Config       OIDCProviderConfig
	oidcProvider *oidc.Provider
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// NewOIDCProvider creates a new OIDC provider, performing discovery against the issuer.
func NewOIDCProvider(ctx context.Context, cfg OIDCProviderConfig, callbackBaseURL string) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discovering OIDC provider %s: %w", cfg.Slug, err)
	}
	oauth2Cfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  callbackBaseURL + "/auth/oidc/" + cfg.Slug + "/callback",
		Scopes:       cfg.Scopes,
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})
	return &OIDCProvider{
		Config:       cfg,
		oidcProvider: provider,
		oauth2Config: oauth2Cfg,
		verifier:     verifier,
	}, nil
}
