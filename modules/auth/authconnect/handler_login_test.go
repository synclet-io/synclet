package authconnect

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/require"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	workspacev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1"
	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// memStorage is an in-memory Storage backing users and refresh tokens.
type memStorage struct {
	authservice.Storage

	users []*authservice.User
}

func (s *memStorage) Users() authservice.UsersStorage { return &memUsersStorage{parent: s} }

func (s *memStorage) RefreshTokens() authservice.RefreshTokensStorage {
	return &memRefreshTokensStorage{}
}

type memUsersStorage struct {
	authservice.UsersStorage

	parent *memStorage
}

func (s *memUsersStorage) Create(_ context.Context, u *authservice.User) (*authservice.User, error) {
	s.parent.users = append(s.parent.users, u)

	return u, nil
}

func (s *memUsersStorage) First(_ context.Context, flt *authservice.UserFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*authservice.User, error) {
	for _, u := range s.parent.users {
		if eq, ok := flt.ID.(*filter.EqualsFilter[uuid.UUID]); ok && eq.Value != u.ID {
			continue
		}

		if eq, ok := flt.Email.(*filter.EqualsFilter[string]); ok && eq.Value != u.Email {
			continue
		}

		return u, nil
	}

	return nil, authservice.ErrUserNotFound
}

type memRefreshTokensStorage struct {
	authservice.RefreshTokensStorage
}

func (s *memRefreshTokensStorage) Create(_ context.Context, t *authservice.RefreshToken) (*authservice.RefreshToken, error) {
	return t, nil
}

type nopAudit struct{}

func (nopAudit) Record(context.Context, authservice.AuditEvent) {}

type membership struct {
	id, name string
	role     workspacev1.MemberRole
}

func membershipsOf(ws []*authv1.WorkspaceMembership) []membership {
	got := make([]membership, 0, len(ws))
	for _, w := range ws {
		got = append(got, membership{w.GetWorkspaceId(), w.GetWorkspaceName(), w.GetRole()})
	}

	return got
}

func newPasswordAuthHandler(storage *memStorage, memberships func(userID uuid.UUID) []authservice.WorkspaceMembership) *Handler {
	cfg := authservice.Config{
		JWTSecret:           "test-secret-test-secret-test-secret",
		AccessTokenTTL:      time.Hour,
		RefreshTokenTTL:     time.Hour,
		RegistrationEnabled: true,
		MinPasswordLength:   8,
	}
	login := authservice.NewLogin(storage, cfg, nopAudit{})
	getUserByID := authservice.NewGetUserByID(storage)
	listUserWorkspaces := authservice.NewListUserWorkspaces(funcMembershipLister(memberships))

	return &Handler{
		registerAndLogin:  authservice.NewRegisterAndLogin(authservice.NewRegister(storage, cfg), login, nil, listUserWorkspaces, nopAudit{}, cfg),
		loginWithUserInfo: authservice.NewLoginWithUserInfo(login, authservice.NewValidateAccessToken(cfg), getUserByID, listUserWorkspaces),
	}
}

type funcMembershipLister func(userID uuid.UUID) []authservice.WorkspaceMembership

func (f funcMembershipLister) ListForUser(_ context.Context, userID uuid.UUID) ([]authservice.WorkspaceMembership, error) {
	return f(userID), nil
}

func TestRegisterAndLogin_ReturnWorkspaceMemberships(t *testing.T) {
	ctx := context.Background()
	wsAdmin, wsViewer := uuid.New(), uuid.New()
	storage := &memStorage{}

	h := newPasswordAuthHandler(storage, func(userID uuid.UUID) []authservice.WorkspaceMembership {
		if len(storage.users) == 0 || storage.users[0].ID != userID {
			return nil
		}

		return []authservice.WorkspaceMembership{
			{WorkspaceID: wsAdmin, WorkspaceName: "Admin WS", Role: authservice.WorkspaceRoleAdmin},
			{WorkspaceID: wsViewer, WorkspaceName: "Viewer WS", Role: authservice.WorkspaceRoleViewer},
		}
	})

	want := []membership{
		{wsAdmin.String(), "Admin WS", workspacev1.MemberRole_MEMBER_ROLE_ADMIN},
		{wsViewer.String(), "Viewer WS", workspacev1.MemberRole_MEMBER_ROLE_VIEWER},
	}

	regResp, err := h.Register(ctx, connect.NewRequest(&authv1.RegisterRequest{
		Email: "a@example.com", Password: "correct-horse-battery", Name: "A",
	}))
	require.NoError(t, err)
	require.Equal(t, want, membershipsOf(regResp.Msg.GetWorkspaces()))

	loginResp, err := h.Login(ctx, connect.NewRequest(&authv1.LoginRequest{
		Email: "a@example.com", Password: "correct-horse-battery",
	}))
	require.NoError(t, err)
	require.Equal(t, storage.users[0].ID.String(), loginResp.Msg.GetUser().GetId())
	require.Equal(t, want, membershipsOf(loginResp.Msg.GetWorkspaces()))
}
