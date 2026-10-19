package authadapt

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

type capturingWriter struct {
	events []auditservice.RecordAuditEventParams
}

func (w *capturingWriter) Execute(_ context.Context, params auditservice.RecordAuditEventParams) {
	w.events = append(w.events, params)
}

type fakeWorkspacesLister struct {
	byUser map[uuid.UUID][]uuid.UUID
	err    error
	calls  int
}

func (l *fakeWorkspacesLister) Execute(_ context.Context, userID uuid.UUID) ([]workspaceservice.UserWorkspace, error) {
	l.calls++
	if l.err != nil {
		return nil, l.err
	}

	out := make([]workspaceservice.UserWorkspace, 0, len(l.byUser[userID]))
	for _, id := range l.byUser[userID] {
		out = append(out, workspaceservice.UserWorkspace{Workspace: &workspaceservice.Workspace{ID: id}})
	}

	return out, nil
}

type noopDelegate struct{}

func (noopDelegate) Info(context.Context, string, ...interface{})         {}
func (noopDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopDelegate) Named(string) logging.Delegate                      { return d }
func (d noopDelegate) SkipCallers(int) logging.Delegate                   { return d }

func newTestRecorder(lister *fakeWorkspacesLister) (*AuditRecorder, *capturingWriter) {
	writer := &capturingWriter{}

	return &AuditRecorder{
		record:     writer,
		workspaces: lister,
		logger:     &logging.Logger{Delegate: noopDelegate{}},
	}, writer
}

func TestAuditRecorder_FansOutUserEventToEveryWorkspace(t *testing.T) {
	userID := uuid.New()
	ws1, ws2 := uuid.New(), uuid.New()
	recorder, writer := newTestRecorder(&fakeWorkspacesLister{byUser: map[uuid.UUID][]uuid.UUID{userID: {ws1, ws2}}})

	recorder.Record(context.Background(), authservice.AuditEvent{
		ActorID:       userID,
		ActorLabel:    "a@example.com",
		Action:        "login",
		ResourceType:  "user",
		ResourceID:    userID,
		ResourceLabel: "a@example.com",
	})

	require.Len(t, writer.events, 2)
	assert.Equal(t, ws1, writer.events[0].WorkspaceID)
	assert.Equal(t, ws2, writer.events[1].WorkspaceID)

	for _, event := range writer.events {
		assert.Equal(t, auditservice.ActorTypeUser, event.ActorType)
		assert.Equal(t, userID, event.ActorID)
		assert.Equal(t, "a@example.com", event.ActorLabel)
		assert.Equal(t, auditservice.ActionLogin, event.Action)
		assert.Equal(t, auditservice.ResourceTypeUser, event.ResourceType)
		assert.Equal(t, userID, event.ResourceID)
	}
}

func TestAuditRecorder_UserWithoutWorkspacesRecordsPendingEvent(t *testing.T) {
	recorder, writer := newTestRecorder(&fakeWorkspacesLister{})

	userID := uuid.New()
	recorder.Record(context.Background(), authservice.AuditEvent{
		ActorID:       userID,
		ActorLabel:    "new@example.com",
		Action:        "create",
		ResourceType:  "user",
		ResourceID:    userID,
		ResourceLabel: "new@example.com",
	})

	require.Len(t, writer.events, 1)
	assert.Equal(t, uuid.Nil, writer.events[0].WorkspaceID)
	assert.Equal(t, auditservice.ActorTypeUser, writer.events[0].ActorType)
	assert.Equal(t, userID, writer.events[0].ActorID)
	assert.Equal(t, auditservice.ActionCreate, writer.events[0].Action)
	assert.Equal(t, auditservice.ResourceTypeUser, writer.events[0].ResourceType)
}

func TestAuditRecorder_ListingFailureRecordsNothing(t *testing.T) {
	recorder, writer := newTestRecorder(&fakeWorkspacesLister{err: errors.New("db down")})

	userID := uuid.New()
	recorder.Record(context.Background(), authservice.AuditEvent{
		ActorID:      userID,
		Action:       "update",
		ResourceType: "user",
		ResourceID:   userID,
	})

	assert.Empty(t, writer.events)
}

func TestAuditRecorder_WorkspaceScopedEventSkipsFanOut(t *testing.T) {
	lister := &fakeWorkspacesLister{}
	recorder, writer := newTestRecorder(lister)

	wsID, userID, keyID := uuid.New(), uuid.New(), uuid.New()
	recorder.Record(context.Background(), authservice.AuditEvent{
		WorkspaceID:   wsID,
		ActorID:       userID,
		Action:        "create",
		ResourceType:  "api_key",
		ResourceID:    keyID,
		ResourceLabel: "ci",
	})

	require.Len(t, writer.events, 1)
	assert.Zero(t, lister.calls)
	assert.Equal(t, wsID, writer.events[0].WorkspaceID)
	assert.Equal(t, auditservice.ActionCreate, writer.events[0].Action)
	assert.Equal(t, auditservice.ResourceTypeAPIKey, writer.events[0].ResourceType)
	assert.Equal(t, keyID, writer.events[0].ResourceID)
}

func TestAuditRecorder_NoActorFallsBackToSystem(t *testing.T) {
	recorder, writer := newTestRecorder(&fakeWorkspacesLister{})

	wsID := uuid.New()
	recorder.Record(context.Background(), authservice.AuditEvent{
		WorkspaceID:  wsID,
		Action:       "delete",
		ResourceType: "api_key",
		ResourceID:   uuid.New(),
	})

	require.Len(t, writer.events, 1)
	assert.Equal(t, auditservice.ActorTypeSystem, writer.events[0].ActorType)
	assert.Equal(t, uuid.Nil, writer.events[0].ActorID)
}
