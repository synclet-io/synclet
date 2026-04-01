package authservice

import (
	"fmt"
	"strings"
)

// OIDCConfig holds the top-level OIDC configuration.
type OIDCConfig struct {
	Providers       []OIDCProviderConfig
	CallbackBaseURL string // OIDC_CALLBACK_BASE_URL (e.g. "https://synclet.mycompany.com")
}

// OIDCProviderConfig holds per-provider OIDC configuration parsed from env vars.
type OIDCProviderConfig struct {
	Slug           string
	DisplayName    string
	Issuer         string
	ClientID       string
	ClientSecret   string
	Scopes         []string
	RoleClaim      string            // e.g. "groups" or "roles"
	RoleMapping    map[string]string // workspace role -> claim value (e.g. "admin" -> "synclet-admins")
	DefaultRole    string            // default: "viewer"
	BoundClaims    map[string]string // claim_name -> required_value
	AllowedDomains []string          // email domain restrictions
	AutoCreateUser bool              // default: true
}

// ValidateBoundClaims checks that all bound claims are present in the token claims.
func ValidateBoundClaims(boundClaims map[string]string, tokenClaims map[string]interface{}) error {
	for claimName, requiredValue := range boundClaims {
		actual, ok := tokenClaims[claimName]
		if !ok {
			return fmt.Errorf("bound claim %q not present in token", claimName)
		}

		actualStr := fmt.Sprintf("%v", actual)
		if actualStr != requiredValue {
			return fmt.Errorf("bound claim %q: expected %q, got %q", claimName, requiredValue, actualStr)
		}
	}

	return nil
}

// ValidateEmailDomain checks email domain against allowed list.
// Requires email_verified=true when allowed domains are configured.
func ValidateEmailDomain(email string, emailVerified bool, allowedDomains []string) error {
	if len(allowedDomains) == 0 {
		return nil
	}

	if !emailVerified {
		return fmt.Errorf("email not verified by provider")
	}

	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}

	domain := strings.ToLower(parts[1])
	for _, allowed := range allowedDomains {
		if strings.ToLower(allowed) == domain {
			return nil
		}
	}

	return fmt.Errorf("email domain %q not in allowed list", domain)
}

// MapRole extracts workspace role from OIDC claims using the configured role claim and mapping.
func MapRole(roleClaim string, roleMapping map[string]string, defaultRole string, tokenClaims map[string]interface{}) string {
	if roleClaim == "" || roleMapping == nil {
		return defaultRole
	}

	raw, ok := tokenClaims[roleClaim]
	if !ok {
		return defaultRole
	}
	// Handle both string and []interface{} (JSON array) claim types.
	var claimValues []string

	switch val := raw.(type) {
	case string:
		claimValues = []string{val}
	case []interface{}:
		for _, item := range val {
			if s, ok := item.(string); ok {
				claimValues = append(claimValues, s)
			}
		}
	case []string:
		claimValues = val
	default:
		return defaultRole
	}
	// Check role mapping in priority order: admin > editor > viewer.
	for _, role := range []string{"admin", "editor", "viewer"} {
		expectedClaim, ok := roleMapping[role]
		if !ok {
			continue
		}

		for _, cv := range claimValues {
			if cv == expectedClaim {
				return role
			}
		}
	}

	return defaultRole
}
