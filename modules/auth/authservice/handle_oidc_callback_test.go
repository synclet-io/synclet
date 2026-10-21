package authservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

type applyCall struct {
	ctxUserID    uuid.UUID
	userID       uuid.UUID
	role         string
	syncExisting bool
}

type fakeRoleApplier struct {
	calls []applyCall
}

func (f *fakeRoleApplier) Apply(ctx context.Context, userID uuid.UUID, role string, syncExisting bool) error {
	ctxUserID, _ := connectutil.UserIDFromContext(ctx)
	f.calls = append(f.calls, applyCall{ctxUserID: ctxUserID, userID: userID, role: role, syncExisting: syncExisting})

	return nil
}

func TestHandleOIDCCallback_ApplyDefaultWorkspaceRole(t *testing.T) {
	adminClaims := map[string]interface{}{"groups": []interface{}{"staff", "synclet-admins"}}
	mapping := map[string]string{"admin": "synclet-admins", "editor": "synclet-editors"}

	tests := []struct {
		name            string
		singleWorkspace bool
		cfg             OIDCProviderConfig
		claims          map[string]interface{}
		wantCall        bool
		wantRole        string
		wantSync        bool
	}{
		{
			name:            "claim set, sync on login",
			singleWorkspace: true,
			cfg:             OIDCProviderConfig{RoleClaim: "groups", RoleMapping: mapping, DefaultRole: "viewer", RoleSync: RoleSyncLogin},
			claims:          adminClaims,
			wantCall:        true,
			wantRole:        "admin",
			wantSync:        true,
		},
		{
			name:            "claim set, sync on first login",
			singleWorkspace: true,
			cfg:             OIDCProviderConfig{RoleClaim: "groups", RoleMapping: mapping, DefaultRole: "viewer", RoleSync: RoleSyncFirstLogin},
			claims:          adminClaims,
			wantCall:        true,
			wantRole:        "admin",
			wantSync:        false,
		},
		{
			name:            "claim set, no value matches",
			singleWorkspace: true,
			cfg:             OIDCProviderConfig{RoleClaim: "groups", RoleMapping: mapping, DefaultRole: "editor", RoleSync: RoleSyncLogin},
			claims:          map[string]interface{}{"groups": []interface{}{"staff"}},
			wantCall:        true,
			wantRole:        "editor",
			wantSync:        true,
		},
		{
			name:            "claim unset",
			singleWorkspace: true,
			cfg:             OIDCProviderConfig{DefaultRole: "viewer", RoleSync: RoleSyncLogin},
			claims:          adminClaims,
			wantCall:        true,
			wantRole:        "viewer",
			wantSync:        false,
		},
		{
			name:     "multi-workspace mode",
			cfg:      OIDCProviderConfig{RoleClaim: "groups", RoleMapping: mapping, DefaultRole: "viewer", RoleSync: RoleSyncLogin},
			claims:   adminClaims,
			wantCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applier := &fakeRoleApplier{}
			uc := &HandleOIDCCallback{
				config:      Config{SingleWorkspaceMode: tt.singleWorkspace},
				roleApplier: applier,
			}
			userID := uuid.New()

			require.NoError(t, uc.applyDefaultWorkspaceRole(context.Background(), tt.cfg, userID, tt.claims))

			if !tt.wantCall {
				assert.Empty(t, applier.calls)

				return
			}

			require.Len(t, applier.calls, 1)
			call := applier.calls[0]
			assert.Equal(t, userID, call.userID)
			assert.Equal(t, userID, call.ctxUserID)
			assert.Equal(t, tt.wantRole, call.role)
			assert.Equal(t, tt.wantSync, call.syncExisting)
		})
	}
}
