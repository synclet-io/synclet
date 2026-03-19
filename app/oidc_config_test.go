package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOIDCProviderConfigs(t *testing.T) {
	t.Setenv("OIDC_GOOGLE_ISSUER", "https://accounts.google.com")
	t.Setenv("OIDC_GOOGLE_CLIENT_ID", "google-client-id")
	t.Setenv("OIDC_GOOGLE_CLIENT_SECRET", "google-secret")
	t.Setenv("OIDC_GOOGLE_DISPLAY_NAME", "Google")

	t.Setenv("OIDC_OKTA_ISSUER", "https://myorg.okta.com")
	t.Setenv("OIDC_OKTA_CLIENT_ID", "okta-client-id")
	t.Setenv("OIDC_OKTA_CLIENT_SECRET", "okta-secret")
	t.Setenv("OIDC_OKTA_SCOPES", "openid,profile")
	t.Setenv("OIDC_OKTA_DEFAULT_ROLE", "editor")
	t.Setenv("OIDC_OKTA_ROLE_CLAIM", "groups")
	t.Setenv("OIDC_OKTA_ROLE_MAP_ADMIN", "synclet-admins")
	t.Setenv("OIDC_OKTA_ALLOWED_DOMAINS", "mycompany.com,partner.com")
	t.Setenv("OIDC_OKTA_BOUND_CLAIM_DEPARTMENT", "engineering")

	configs, err := parseOIDCProviderConfigs("google,okta")
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// Google provider.
	g := configs[0]
	assert.Equal(t, "google", g.Slug)
	assert.Equal(t, "Google", g.DisplayName)
	assert.Equal(t, "https://accounts.google.com", g.Issuer)
	assert.Equal(t, "google-client-id", g.ClientID)
	assert.Equal(t, "google-secret", g.ClientSecret)
	assert.Equal(t, []string{"openid", "profile", "email"}, g.Scopes) // default
	assert.Equal(t, "viewer", g.DefaultRole)                          // default
	assert.True(t, g.AutoCreateUser)

	// Okta provider.
	o := configs[1]
	assert.Equal(t, "okta", o.Slug)
	assert.Equal(t, "https://myorg.okta.com", o.Issuer)
	assert.Equal(t, []string{"openid", "profile"}, o.Scopes)
	assert.Equal(t, "editor", o.DefaultRole)
	assert.Equal(t, "groups", o.RoleClaim)
	assert.Equal(t, map[string]string{"admin": "synclet-admins"}, o.RoleMapping)
	assert.Equal(t, []string{"mycompany.com", "partner.com"}, o.AllowedDomains)
	assert.Equal(t, map[string]string{"department": "engineering"}, o.BoundClaims)
}

func TestParseOIDCProviderConfigs_Empty(t *testing.T) {
	configs, err := parseOIDCProviderConfigs("")
	require.NoError(t, err)
	assert.Nil(t, configs)
}

func TestParseOIDCProviderConfigs_Defaults(t *testing.T) {
	t.Setenv("OIDC_DEV_ISSUER", "https://dev.example.com")
	t.Setenv("OIDC_DEV_CLIENT_ID", "dev-id")
	t.Setenv("OIDC_DEV_CLIENT_SECRET", "dev-secret")

	configs, err := parseOIDCProviderConfigs("dev")
	require.NoError(t, err)
	require.Len(t, configs, 1)

	assert.Equal(t, []string{"openid", "profile", "email"}, configs[0].Scopes)
	assert.Equal(t, "viewer", configs[0].DefaultRole)
	assert.Equal(t, "dev", configs[0].DisplayName) // fallback to slug
	assert.True(t, configs[0].AutoCreateUser)
}
