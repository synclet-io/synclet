package connectutil

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	optionsv1 "github.com/synclet-io/synclet/gen/proto/synclet/options/v1"
)

// MembershipChecker verifies workspace membership for authorization.
type MembershipChecker interface {
	IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
	GetMemberRole(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
}

// RoleInterceptor enforces role-based authorization by reading proto method options.
// RPCs annotated with required_role are checked against the caller's workspace role.
// RPCs without the annotation pass through (no role check).
type RoleInterceptor struct {
	roleMap map[string]optionsv1.RequiredRole
	checker MembershipChecker
}

// NewRoleInterceptor creates a role interceptor that builds its role map from proto registry.
func NewRoleInterceptor(checker MembershipChecker) *RoleInterceptor {
	return &RoleInterceptor{
		roleMap: buildRoleMap(),
		checker: checker,
	}
}

// buildRoleMap iterates all registered proto service methods and extracts required_role options.
func buildRoleMap() map[string]optionsv1.RequiredRole {
	roles := make(map[string]optionsv1.RequiredRole)

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := range fd.Services().Len() {
			svcDesc := fd.Services().Get(i)
			for j := range svcDesc.Methods().Len() {
				methodDesc := svcDesc.Methods().Get(j)

				opts := methodDesc.Options()
				if opts == nil {
					continue
				}

				if !proto.HasExtension(opts, optionsv1.E_RequiredRole) {
					continue
				}

				role := proto.GetExtension(opts, optionsv1.E_RequiredRole).(optionsv1.RequiredRole)
				if role != optionsv1.RequiredRole_REQUIRED_ROLE_UNSPECIFIED {
					procedure := "/" + string(svcDesc.FullName()) + "/" + string(methodDesc.Name())
					roles[procedure] = role
				}
			}
		}

		return true
	})

	return roles
}

func (i *RoleInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := i.checkRole(ctx, req.Spec().Procedure); err != nil {
			return nil, err
		}

		return next(ctx, req)
	}
}

func (i *RoleInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *RoleInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := i.checkRole(ctx, conn.Spec().Procedure); err != nil {
			return err
		}

		return next(ctx, conn)
	}
}

func (i *RoleInterceptor) checkRole(ctx context.Context, procedure string) error {
	requiredRole, exists := i.roleMap[procedure]
	if !exists {
		return nil // No annotation = no role check.
	}

	wsID, err := WorkspaceIDFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, nil)
	}

	userID, err := UserIDFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, nil)
	}

	roleStr, err := i.checker.GetMemberRole(ctx, wsID, userID)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, nil)
	}

	if !meetsMinimumRole(roleStr, requiredRole) {
		return connect.NewError(connect.CodePermissionDenied, nil)
	}

	return nil
}

// meetsMinimumRole compares the actual role string against the required proto role.
// Role hierarchy: Viewer(1) < Editor(2) < Admin(3).
// Role strings are PascalCase ("Admin", "Editor", "Viewer") as returned
// by MemberRole.String() via MembershipChecker.GetMemberRole().
func meetsMinimumRole(actual string, required optionsv1.RequiredRole) bool {
	roleLevels := map[string]int{
		"Admin":  3,
		"Editor": 2,
		"Viewer": 1,
	}
	requiredLevels := map[optionsv1.RequiredRole]int{
		optionsv1.RequiredRole_REQUIRED_ROLE_VIEWER: 1,
		optionsv1.RequiredRole_REQUIRED_ROLE_EDITOR: 2,
		optionsv1.RequiredRole_REQUIRED_ROLE_ADMIN:  3,
	}

	return roleLevels[actual] >= requiredLevels[required]
}
