package workspaceservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingLogDelegate struct {
	mu     *sync.Mutex
	fields map[string]interface{}
	warns  *[]capturedWarn
}

type capturedWarn struct {
	msg    string
	fields map[string]interface{}
}

func newCapturingLogger() (*logging.Logger, *[]capturedWarn) {
	warns := &[]capturedWarn{}

	return &logging.Logger{Delegate: capturingLogDelegate{mu: &sync.Mutex{}, fields: map[string]interface{}{}, warns: warns}}, warns
}

func (d capturingLogDelegate) with(fields map[string]interface{}) capturingLogDelegate {
	merged := make(map[string]interface{}, len(d.fields)+len(fields))
	for k, v := range d.fields {
		merged[k] = v
	}

	for k, v := range fields {
		merged[k] = v
	}

	return capturingLogDelegate{mu: d.mu, fields: merged, warns: d.warns}
}

func (d capturingLogDelegate) Info(context.Context, string, ...interface{})  {}
func (d capturingLogDelegate) Debug(context.Context, string, ...interface{}) {}
func (d capturingLogDelegate) Error(context.Context, string, ...interface{}) {}

func (d capturingLogDelegate) Warn(_ context.Context, msg string, args ...interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	*d.warns = append(*d.warns, capturedWarn{msg: fmt.Sprintf(msg, args...), fields: d.fields})
}

func (d capturingLogDelegate) WithFields(fields map[string]interface{}) logging.Delegate {
	return d.with(fields)
}

func (d capturingLogDelegate) WithField(key string, value interface{}) logging.Delegate {
	return d.with(map[string]interface{}{key: value})
}

func (d capturingLogDelegate) WithError(err error) logging.Delegate {
	return d.with(map[string]interface{}{"error": err})
}

func (d capturingLogDelegate) Named(string) logging.Delegate    { return d }
func (d capturingLogDelegate) SkipCallers(int) logging.Delegate { return d }

func TestApplyDefaultWorkspaceRole(t *testing.T) {
	tests := []struct {
		name         string
		otherMembers []MemberRole
		existing     *MemberRole
		role         MemberRole
		syncExisting bool
		wantRole     MemberRole
		wantAudit    bool
		wantWarn     bool
	}{
		{name: "first member", role: MemberRoleEditor, syncExisting: true, wantRole: MemberRoleAdmin},
		{name: "later member", otherMembers: []MemberRole{MemberRoleAdmin}, role: MemberRoleEditor, wantRole: MemberRoleEditor},
		{name: "later member with sync", otherMembers: []MemberRole{MemberRoleAdmin}, role: MemberRoleViewer, syncExisting: true, wantRole: MemberRoleViewer},
		{name: "existing, sync", otherMembers: []MemberRole{MemberRoleAdmin}, existing: rolePtr(MemberRoleViewer), role: MemberRoleAdmin, syncExisting: true, wantRole: MemberRoleAdmin, wantAudit: true},
		{name: "existing, no sync", otherMembers: []MemberRole{MemberRoleAdmin}, existing: rolePtr(MemberRoleViewer), role: MemberRoleAdmin, wantRole: MemberRoleViewer},
		{name: "unchanged", otherMembers: []MemberRole{MemberRoleAdmin}, existing: rolePtr(MemberRoleEditor), role: MemberRoleEditor, syncExisting: true, wantRole: MemberRoleEditor},
		{name: "last admin", otherMembers: []MemberRole{MemberRoleEditor}, existing: rolePtr(MemberRoleAdmin), role: MemberRoleViewer, syncExisting: true, wantRole: MemberRoleAdmin, wantWarn: true},
		{name: "one of two admins", otherMembers: []MemberRole{MemberRoleAdmin}, existing: rolePtr(MemberRoleAdmin), role: MemberRoleViewer, syncExisting: true, wantRole: MemberRoleViewer, wantAudit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			workspaceID := uuid.New()
			userID := uuid.New()

			store := newFakeStorage()
			store.seedWorkspace(&Workspace{ID: workspaceID, Name: "Default", Slug: "default"})

			for _, role := range tt.otherMembers {
				store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: uuid.New(), Role: role})
			}

			memberID := uuid.New()
			if tt.existing != nil {
				store.seedMember(&WorkspaceMember{ID: memberID, WorkspaceID: workspaceID, UserID: userID, Role: *tt.existing})
			}

			recorder := &capturingAuditRecorder{}
			logger, warns := newCapturingLogger()

			err := NewApplyDefaultWorkspaceRole(store, recorder, logger).Execute(ctx, userID, tt.role, tt.syncExisting)
			require.NoError(t, err)

			members, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{UserID: equalsUUID(userID)})
			require.NoError(t, err)
			require.Len(t, members, 1)
			assert.Equal(t, workspaceID, members[0].WorkspaceID)
			assert.Equal(t, tt.wantRole, members[0].Role)

			if tt.wantAudit {
				require.Len(t, recorder.events, 1)
				event := recorder.events[0]
				assert.Equal(t, workspaceID, event.WorkspaceID)
				assert.Equal(t, "update", event.Action)
				assert.Equal(t, "workspace_member", event.ResourceType)
				assert.Equal(t, memberID, event.ResourceID)
				assert.Equal(t, map[string]any{"role": tt.existing.String(), "user_id": userID.String()}, event.Before)
				assert.Equal(t, map[string]any{"role": tt.role.String(), "user_id": userID.String()}, event.After)
			} else {
				assert.Empty(t, recorder.events)
			}

			if tt.wantWarn {
				require.Len(t, *warns, 1)
				warn := (*warns)[0]
				assert.Equal(t, userID.String(), fmt.Sprint(warn.fields["user_id"]))
				assert.Equal(t, tt.role.String(), fmt.Sprint(warn.fields["requested_role"]))
			} else {
				assert.Empty(t, *warns)
			}
		})
	}
}

func rolePtr(r MemberRole) *MemberRole {
	return &r
}

type racingMembersStorage struct {
	WorkspaceMembersStorage
}

func (s racingMembersStorage) Find(context.Context, *WorkspaceMemberFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*WorkspaceMember, error) {
	return nil, nil
}

func (s racingMembersStorage) Create(context.Context, *WorkspaceMember) (*WorkspaceMember, error) {
	return nil, errors.Join(ErrWorkspaceMemberAlreadyExists, errors.New(`duplicate key value violates unique constraint (SQLSTATE 23505)`))
}

type racingStorage struct {
	*fakeStorage
}

func (s racingStorage) WorkspaceMembers() WorkspaceMembersStorage {
	return racingMembersStorage{WorkspaceMembersStorage: s.fakeStorage.WorkspaceMembers()}
}

func TestApplyDefaultWorkspaceRole_ConcurrentJoinIsIdempotent(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()

	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: workspaceID, Name: "Default", Slug: "default"})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: uuid.New(), Role: MemberRoleAdmin})

	recorder := &capturingAuditRecorder{}
	logger, warns := newCapturingLogger()

	err := NewApplyDefaultWorkspaceRole(racingStorage{store}, recorder, logger).Execute(ctx, uuid.New(), MemberRoleEditor, true)
	require.NoError(t, err)
	assert.Empty(t, recorder.events)
	assert.Empty(t, *warns)
}
