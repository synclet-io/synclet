package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// parseOIDCConfig parses OIDC configuration from environment variables.
// Returns nil if OIDC_PROVIDERS is empty (OIDC disabled).
func parseOIDCConfig() (*authservice.OIDCConfig, error) {
	providersEnv := os.Getenv("OIDC_PROVIDERS")
	if providersEnv == "" {
		return nil, nil
	}

	callbackBase := os.Getenv("OIDC_CALLBACK_BASE_URL")
	if callbackBase == "" {
		return nil, errors.New("OIDC_CALLBACK_BASE_URL is required when OIDC_PROVIDERS is set")
	}

	providers, err := parseOIDCProviderConfigs(providersEnv)
	if err != nil {
		return nil, err
	}

	return &authservice.OIDCConfig{Providers: providers, CallbackBaseURL: callbackBase}, nil
}

// parseOIDCProviderConfigs parses comma-separated provider slugs and loads
// per-provider config from OIDC_{SLUG}_* env vars.
func parseOIDCProviderConfigs(providersList string) ([]authservice.OIDCProviderConfig, error) {
	if providersList == "" {
		return nil, nil
	}

	slugs := strings.Split(providersList, ",")
	var configs []authservice.OIDCProviderConfig

	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}

		cfg, err := parseOIDCProviderConfig(slug)
		if err != nil {
			return nil, fmt.Errorf("provider %s: %w", slug, err)
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

func parseOIDCProviderConfig(slug string) (authservice.OIDCProviderConfig, error) {
	prefix := "OIDC_" + strings.ToUpper(slug) + "_"

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
