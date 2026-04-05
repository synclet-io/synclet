package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// parseOIDCProviderConfigs parses per-provider OIDC configuration from environment variables.
// Returns nil if OIDCProviders is empty (OIDC disabled).
func parseOIDCProviderConfigs(cfg *authConfig) ([]authservice.OIDCProviderConfig, error) {
	if len(cfg.OIDCProviders) == 0 {
		return nil, nil
	}

	if cfg.OIDCCallbackBaseURL == "" {
		return nil, errors.New("AUTH_OIDC_CALLBACK_BASE_URL is required when AUTH_OIDC_PROVIDERS is set")
	}

	var providers []authservice.OIDCProviderConfig

	for _, slug := range cfg.OIDCProviders {
		providerCfg, err := parseOIDCProviderConfig(slug)
		if err != nil {
			return nil, fmt.Errorf("provider %s: %w", slug, err)
		}

		providers = append(providers, providerCfg)
	}

	return providers, nil
}

func parseOIDCProviderConfig(slug string) (authservice.OIDCProviderConfig, error) {
	prefix := "AUTH_OIDC_" + strings.TrimSpace(strings.ToUpper(slug)) + "_"

	issuer := os.Getenv(prefix + "ISSUER")
	if issuer == "" {
		return authservice.OIDCProviderConfig{}, fmt.Errorf("%sISSUER is required", prefix)
	}

	clientID := os.Getenv(prefix + "CLIENT_ID")
	if clientID == "" {
		return authservice.OIDCProviderConfig{}, fmt.Errorf("%sCLIENT_ID is required", prefix)
	}

	clientSecret := os.Getenv(prefix + "CLIENT_SECRET")
	if clientSecret == "" {
		return authservice.OIDCProviderConfig{}, fmt.Errorf("%sCLIENT_SECRET is required", prefix)
	}

	displayName := os.Getenv(prefix + "DISPLAY_NAME")
	if displayName == "" {
		displayName = slug
	}

	scopes := parseCSV(os.Getenv(prefix + "SCOPES"))
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	defaultRole := os.Getenv(prefix + "DEFAULT_ROLE")
	if defaultRole == "" {
		defaultRole = "viewer"
	}

	autoCreate := os.Getenv(prefix + "AUTO_CREATE_USER")
	autoCreateUser := autoCreate == "" || autoCreate == "true"

	roleMapping := parseOIDCRoleMapping(prefix)
	boundClaims := parseOIDCBoundClaims(prefix)
	allowedDomains := parseCSV(os.Getenv(prefix + "ALLOWED_DOMAINS"))

	return authservice.OIDCProviderConfig{
		Slug:           slug,
		DisplayName:    displayName,
		Issuer:         issuer,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         scopes,
		RoleClaim:      os.Getenv(prefix + "ROLE_CLAIM"),
		RoleMapping:    roleMapping,
		DefaultRole:    defaultRole,
		BoundClaims:    boundClaims,
		AllowedDomains: allowedDomains,
		AutoCreateUser: autoCreateUser,
	}, nil
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}

	return result
}

func parseOIDCRoleMapping(prefix string) map[string]string {
	roleMapping := make(map[string]string)

	for _, role := range []string{"admin", "editor", "viewer"} {
		val := os.Getenv(prefix + "ROLE_MAP_" + strings.ToUpper(role))
		if val != "" {
			roleMapping[role] = val
		}
	}

	if len(roleMapping) == 0 {
		return nil
	}

	return roleMapping
}

func parseOIDCBoundClaims(prefix string) map[string]string {
	boundClaims := make(map[string]string)

	boundPrefix := prefix + "BOUND_CLAIM_"
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, boundPrefix) {
			kv := strings.SplitN(env, "=", 2)
			if len(kv) == 2 {
				claimName := strings.ToLower(strings.TrimPrefix(kv[0], boundPrefix))
				boundClaims[claimName] = kv[1]
			}
		}
	}

	if len(boundClaims) == 0 {
		return nil
	}

	return boundClaims
}
