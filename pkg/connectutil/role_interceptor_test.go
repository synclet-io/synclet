package connectutil

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	optionsv1 "github.com/synclet-io/synclet/gen/proto/synclet/options/v1"
)

// mockMembershipChecker is a test double for MembershipChecker.
type mockMembershipChecker struct {
	isMember    bool
	isMemberErr error
	role        string
	roleErr     error
}

func (m *mockMembershipChecker) IsMember(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isMember, m.isMemberErr
}

func (m *mockMembershipChecker) GetMemberRole(_ context.Context, _, _ uuid.UUID) (string, error) {
	return m.role, m.roleErr
}

// newRoleInterceptorWithMap creates a RoleInterceptor with a pre-built role map for testing.
func newRoleInterceptorWithMap(roleMap map[string]optionsv1.RequiredRole, checker MembershipChecker) *RoleInterceptor {
	return &RoleInterceptor{roleMap: roleMap, checker: checker}
}

func TestRoleInterceptor_checkRole(t *testing.T) {
	procedure := "/synclet.publicapi.test.v1.TestService/TestMethod"

	t.Run("admin annotation with Admin caller passes", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Admin"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_ADMIN},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.NoError(t, err)
	})

	t.Run("admin annotation with Editor caller denied", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Editor"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_ADMIN},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})

	t.Run("admin annotation with Viewer caller denied", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Viewer"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_ADMIN},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})

	t.Run("editor annotation with Editor caller passes", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Editor"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_EDITOR},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.NoError(t, err)
	})

	t.Run("editor annotation with Admin caller passes (hierarchy)", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Admin"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_EDITOR},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.NoError(t, err)
	})

	t.Run("editor annotation with Viewer caller denied", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Viewer"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_EDITOR},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})

	t.Run("viewer annotation with Viewer caller passes", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Viewer"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.NoError(t, err)
	})

	t.Run("viewer annotation with Editor caller passes", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Editor"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.NoError(t, err)
	})

	t.Run("viewer annotation with Admin caller passes", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Admin"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.NoError(t, err)
	})

	t.Run("unannotated RPC passes through", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Viewer"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{}, // empty map
			checker,
		)

		err := interceptor.checkRole(context.Background(), "/some.other.Service/Method")
		require.NoError(t, err)
	})

	t.Run("no workspace ID in context returns permission denied", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Admin"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER},
			checker,
		)

		ctx := ContextWithUserID(context.Background(), uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})

	t.Run("no user ID in context returns permission denied", func(t *testing.T) {
		checker := &mockMembershipChecker{role: "Admin"}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})

	t.Run("GetMemberRole error returns permission denied", func(t *testing.T) {
		checker := &mockMembershipChecker{roleErr: errors.New("db error")}
		interceptor := newRoleInterceptorWithMap(
			map[string]optionsv1.RequiredRole{procedure: optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER},
			checker,
		)

		ctx := ContextWithWorkspaceID(context.Background(), uuid.New())
		ctx = ContextWithUserID(ctx, uuid.New())

		err := interceptor.checkRole(ctx, procedure)
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})
}

func TestMeetsMinimumRole(t *testing.T) {
	t.Run("unknown role string returns false", func(t *testing.T) {
		assert.False(t, meetsMinimumRole("unknown", optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER))
	})

	t.Run("empty role string returns false", func(t *testing.T) {
		assert.False(t, meetsMinimumRole("", optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER))
	})

	t.Run("lowercase role string returns false", func(t *testing.T) {
		// Role strings must be PascalCase as returned by MemberRole.String().
		assert.False(t, meetsMinimumRole("admin", optionsv1.RequiredRole_REQUIRED_ROLE_ADMIN))
	})
}
