package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

func TestParseOIDCProviderConfigs(t *testing.T) {
	t.Setenv("AUTH_OIDC_GOOGLE_ISSUER", "https://accounts.google.com")
	t.Setenv("AUTH_OIDC_GOOGLE_CLIENT_ID", "google-client-id")
	t.Setenv("AUTH_OIDC_GOOGLE_CLIENT_SECRET", "google-secret")
	t.Setenv("AUTH_OIDC_GOOGLE_DISPLAY_NAME", "Google")

	t.Setenv("AUTH_OIDC_OKTA_ISSUER", "https://myorg.okta.com")
	t.Setenv("AUTH_OIDC_OKTA_CLIENT_ID", "okta-client-id")
	t.Setenv("AUTH_OIDC_OKTA_CLIENT_SECRET", "okta-secret")
	t.Setenv("AUTH_OIDC_OKTA_SCOPES", "openid,profile")
	t.Setenv("AUTH_OIDC_OKTA_DEFAULT_ROLE", "editor")
	t.Setenv("AUTH_OIDC_OKTA_ROLE_CLAIM", "groups")
	t.Setenv("AUTH_OIDC_OKTA_ROLE_MAP_ADMIN", "synclet-admins")
	t.Setenv("AUTH_OIDC_OKTA_ALLOWED_DOMAINS", "mycompany.com,partner.com")
	t.Setenv("AUTH_OIDC_OKTA_BOUND_CLAIM_DEPARTMENT", "engineering")

	configs, err := parseOIDCProviderConfigs(&authConfig{
		OIDCProviders:       []string{"google", "okta"},
		OIDCCallbackBaseURL: "https://example.com",
	})
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// Google provider.
	google := configs[0]
	assert.Equal(t, "google", google.Slug)
	assert.Equal(t, "Google", google.DisplayName)
	assert.Equal(t, "https://accounts.google.com", google.Issuer)
	assert.Equal(t, "google-client-id", google.ClientID)
	assert.Equal(t, "google-secret", google.ClientSecret)
	assert.Equal(t, []string{"openid", "profile", "email"}, google.Scopes) // default
	assert.Equal(t, "viewer", google.DefaultRole)                          // default
	assert.True(t, google.AutoCreateUser)

	// Okta provider.
	okta := configs[1]
	assert.Equal(t, "okta", okta.Slug)
	assert.Equal(t, "https://myorg.okta.com", okta.Issuer)
	assert.Equal(t, []string{"openid", "profile"}, okta.Scopes)
	assert.Equal(t, "editor", okta.DefaultRole)
	assert.Equal(t, "groups", okta.RoleClaim)
	assert.Equal(t, map[string]string{"admin": "synclet-admins"}, okta.RoleMapping)
	assert.Equal(t, []string{"mycompany.com", "partner.com"}, okta.AllowedDomains)
	assert.Equal(t, map[string]string{"department": "engineering"}, okta.BoundClaims)
}

func TestParseOIDCProviderConfigs_Empty(t *testing.T) {
	configs, err := parseOIDCProviderConfigs(&authConfig{})
	require.NoError(t, err)
	assert.Nil(t, configs)
}

func TestParseOIDCProviderConfigs_Defaults(t *testing.T) {
	t.Setenv("AUTH_OIDC_DEV_ISSUER", "https://dev.example.com")
	t.Setenv("AUTH_OIDC_DEV_CLIENT_ID", "dev-id")
	t.Setenv("AUTH_OIDC_DEV_CLIENT_SECRET", "dev-secret")

	configs, err := parseOIDCProviderConfigs(&authConfig{
		OIDCProviders:       []string{"dev"},
		OIDCCallbackBaseURL: "https://example.com",
	})
	require.NoError(t, err)
	require.Len(t, configs, 1)

	assert.Equal(t, []string{"openid", "profile", "email"}, configs[0].Scopes)
	assert.Equal(t, "viewer", configs[0].DefaultRole)
	assert.Equal(t, "dev", configs[0].DisplayName) // fallback to slug
	assert.True(t, configs[0].AutoCreateUser)
}

func setOIDCProviderEnv(t *testing.T, slug string) {
	t.Helper()

	t.Setenv("AUTH_OIDC_"+slug+"_ISSUER", "https://idp.example.com")
	t.Setenv("AUTH_OIDC_"+slug+"_CLIENT_ID", "id")
	t.Setenv("AUTH_OIDC_"+slug+"_CLIENT_SECRET", "secret")
}

func TestParseOIDCProviderConfigs_RoleSettings(t *testing.T) {
	tests := []struct {
		name        string
		defaultRole string
		roleSync    string
		wantSync    authservice.RoleSync
		wantErr     string
	}{
		{name: "sync defaults to login", wantSync: authservice.RoleSyncLogin},
		{name: "sync login", roleSync: "login", wantSync: authservice.RoleSyncLogin},
		{name: "sync first_login", roleSync: "first_login", wantSync: authservice.RoleSyncFirstLogin},
		{name: "bad sync", roleSync: "always", wantErr: "AUTH_OIDC_X_ROLE_SYNC"},
		{name: "default role admin", defaultRole: "admin", wantSync: authservice.RoleSyncLogin},
		{name: "bad default role", defaultRole: "owner", wantErr: "AUTH_OIDC_X_DEFAULT_ROLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setOIDCProviderEnv(t, "X")

			if tt.defaultRole != "" {
				t.Setenv("AUTH_OIDC_X_DEFAULT_ROLE", tt.defaultRole)
			}

			if tt.roleSync != "" {
				t.Setenv("AUTH_OIDC_X_ROLE_SYNC", tt.roleSync)
			}

			configs, err := parseOIDCProviderConfigs(&authConfig{
				OIDCProviders:       []string{"x"},
				OIDCCallbackBaseURL: "https://example.com",
			})
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Len(t, configs, 1)
			assert.Equal(t, tt.wantSync, configs[0].RoleSync)
		})
	}
}

func TestOIDCRoleSettingsWarnings(t *testing.T) {
	tests := []struct {
		name            string
		env             map[string]string
		singleWorkspace bool
		wantFor         []string
	}{
		{
			name: "multi with role settings",
			env: map[string]string{
				"AUTH_OIDC_AUTHENTIK_ROLE_CLAIM":     "groups",
				"AUTH_OIDC_AUTHENTIK_ROLE_MAP_ADMIN": "synclet-admins",
				"AUTH_OIDC_GOOGLE_ROLE_SYNC":         "first_login",
				"AUTH_OIDC_OKTA_DEFAULT_ROLE":        "editor",
			},
			wantFor: []string{"authentik", "google", "okta"},
		},
		{
			name: "multi without role settings",
			env:  map[string]string{"AUTH_OIDC_AUTHENTIK_ISSUER": "https://idp.example.com"},
		},
		{
			name:            "single with role settings",
			env:             map[string]string{"AUTH_OIDC_AUTHENTIK_ROLE_CLAIM": "groups"},
			singleWorkspace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			warnings := oidcRoleSettingsWarnings([]string{"authentik", "google", "okta", "dev"}, tt.singleWorkspace)
			require.Len(t, warnings, len(tt.wantFor))

			for i, slug := range tt.wantFor {
				assert.Contains(t, warnings[i], "OIDC provider "+slug+" ")
				assert.Contains(t, warnings[i], "single-workspace mode only")
			}
		})
	}
}
