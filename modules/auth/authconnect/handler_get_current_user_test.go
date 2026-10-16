package authconnect

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/require"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	workspacev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type stubStorage struct {
	authservice.Storage

	user *authservice.User
}

func (s *stubStorage) Users() authservice.UsersStorage { return &stubUsersStorage{user: s.user} }

type stubUsersStorage struct {
	authservice.UsersStorage

	user *authservice.User
}

func (s *stubUsersStorage) First(context.Context, *authservice.UserFilter, ...optionutil.Option[dbutil.SelectOptions]) (*authservice.User, error) {
	return s.user, nil
}

type stubMembershipLister struct {
	memberships map[uuid.UUID][]authservice.WorkspaceMembership
}

func (l *stubMembershipLister) ListForUser(_ context.Context, userID uuid.UUID) ([]authservice.WorkspaceMembership, error) {
	return l.memberships[userID], nil
}

func TestGetCurrentUser_ReturnsEveryMembershipWithRole(t *testing.T) {
	user := &authservice.User{ID: uuid.New(), Email: "a@example.com", Name: "A", CreatedAt: time.Now()}
	wsAdmin, wsEditor, wsViewer := uuid.New(), uuid.New(), uuid.New()
	lister := &stubMembershipLister{memberships: map[uuid.UUID][]authservice.WorkspaceMembership{
		user.ID: {
			{WorkspaceID: wsAdmin, WorkspaceName: "Admin WS", Role: authservice.WorkspaceRoleAdmin},
			{WorkspaceID: wsEditor, WorkspaceName: "Editor WS", Role: authservice.WorkspaceRoleEditor},
			{WorkspaceID: wsViewer, WorkspaceName: "Viewer WS", Role: authservice.WorkspaceRoleViewer},
		},
	}}

	h := &Handler{
		getUserByID:        authservice.NewGetUserByID(&stubStorage{user: user}),
		listUserWorkspaces: authservice.NewListUserWorkspaces(lister),
	}

	ctx := connectutil.ContextWithUserID(context.Background(), user.ID)
	resp, err := h.GetCurrentUser(ctx, connect.NewRequest(&authv1.GetCurrentUserRequest{}))
	require.NoError(t, err)

	require.Equal(t, user.ID.String(), resp.Msg.GetUser().GetId())

	type membership struct {
		id, name string
		role     workspacev1.MemberRole
	}

	got := make([]membership, 0, len(resp.Msg.GetWorkspaces()))
	for _, ws := range resp.Msg.GetWorkspaces() {
		got = append(got, membership{ws.GetWorkspaceId(), ws.GetWorkspaceName(), ws.GetRole()})
	}

	require.Equal(t, []membership{
		{wsAdmin.String(), "Admin WS", workspacev1.MemberRole_MEMBER_ROLE_ADMIN},
		{wsEditor.String(), "Editor WS", workspacev1.MemberRole_MEMBER_ROLE_EDITOR},
		{wsViewer.String(), "Viewer WS", workspacev1.MemberRole_MEMBER_ROLE_VIEWER},
	}, got)
}
