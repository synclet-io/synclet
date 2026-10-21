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

	switch defaultRole {
	case "admin", "editor", "viewer":
	default:
		return authservice.OIDCProviderConfig{}, fmt.Errorf("%sDEFAULT_ROLE must be admin, editor or viewer, got %q", prefix, defaultRole)
	}

	roleSync := authservice.RoleSync(os.Getenv(prefix + "ROLE_SYNC"))
	switch roleSync {
	case "":
		roleSync = authservice.RoleSyncLogin
	case authservice.RoleSyncLogin, authservice.RoleSyncFirstLogin:
	default:
		return authservice.OIDCProviderConfig{}, fmt.Errorf("%sROLE_SYNC must be login or first_login, got %q", prefix, roleSync)
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
		RoleSync:       roleSync,
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

var oidcRoleSettingSuffixes = []string{
	"ROLE_CLAIM",
	"ROLE_MAP_ADMIN",
	"ROLE_MAP_EDITOR",
	"ROLE_MAP_VIEWER",
	"DEFAULT_ROLE",
	"ROLE_SYNC",
}

// oidcRoleSettingsWarnings returns one warning per provider that sets a role
// setting while the server runs in multi-workspace mode, where role settings
// have no effect.
func oidcRoleSettingsWarnings(slugs []string, singleWorkspace bool) []string {
	if singleWorkspace {
		return nil
	}

	var warnings []string

	for _, slug := range slugs {
		prefix := "AUTH_OIDC_" + strings.TrimSpace(strings.ToUpper(slug)) + "_"

		var set []string

		for _, suffix := range oidcRoleSettingSuffixes {
			if _, ok := os.LookupEnv(prefix + suffix); ok {
				set = append(set, prefix+suffix)
			}
		}

		if len(set) > 0 {
			warnings = append(warnings, fmt.Sprintf(
				"OIDC provider %s sets %s, but OIDC role settings apply to single-workspace mode only and are ignored in multi-workspace mode",
				slug, strings.Join(set, ", "),
			))
		}
	}

	return warnings
}
