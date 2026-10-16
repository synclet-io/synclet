package auditservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

var testConfig = Config{DefaultRetentionDays: 90, MaxRetentionDays: 365}

func TestUpdateAuditRetention_RejectsOutOfBounds(t *testing.T) {
	storage := newFakeStorage()
	uc := NewUpdateAuditRetention(storage, testConfig, NewRecordAuditEvent(storage, nil))
	workspaceID := uuid.New()

	for _, days := range []int{0, testConfig.MaxRetentionDays + 1} {
		_, err := uc.Execute(context.Background(), UpdateAuditRetentionParams{WorkspaceID: workspaceID, RetentionDays: days})

		var validationErr *ValidationError
		require.ErrorAs(t, err, &validationErr, "days=%d", days)
	}

	assert.Empty(t, storage.retention)
	assert.Empty(t, storage.outbox, "rejected updates must not be audited")
}

func TestUpdateAuditRetention_AuditsChangeByUser(t *testing.T) {
	storage := newFakeStorage()
	uc := NewUpdateAuditRetention(storage, testConfig, NewRecordAuditEvent(storage, nil))
	workspaceID := uuid.New()
	userID := uuid.New()
	storage.retention = []*AuditRetentionSetting{{WorkspaceID: workspaceID, RetentionDays: 30}}
	ctx := connectutil.ContextWithUserID(context.Background(), userID)

	_, err := uc.Execute(ctx, UpdateAuditRetentionParams{WorkspaceID: workspaceID, RetentionDays: 60})
	require.NoError(t, err)

	event := singleRecordedAuditEvent(t, storage)
	assert.Equal(t, workspaceID, event.WorkspaceID)
	assert.Equal(t, ActorTypeUser, event.ActorType)
	assert.Equal(t, userID, event.ActorID)
	assert.Equal(t, ActionUpdate, event.Action)
	assert.Equal(t, ResourceTypeWorkspace, event.ResourceType)
	assert.Equal(t, workspaceID, event.ResourceID)
	assert.Equal(t, "audit retention", event.ResourceLabel)
	assert.JSONEq(t, `[{"path":"audit_retention_days","before":30,"after":60}]`, event.DiffJSON)
}

func TestUpdateAuditRetention_AuditsChangeBySystemWithDefaultBefore(t *testing.T) {
	storage := newFakeStorage()
	uc := NewUpdateAuditRetention(storage, testConfig, NewRecordAuditEvent(storage, nil))
	workspaceID := uuid.New()

	_, err := uc.Execute(context.Background(), UpdateAuditRetentionParams{WorkspaceID: workspaceID, RetentionDays: 45})
	require.NoError(t, err)

	event := singleRecordedAuditEvent(t, storage)
	assert.Equal(t, ActorTypeSystem, event.ActorType)
	assert.Equal(t, uuid.Nil, event.ActorID)
	assert.JSONEq(t, `[{"path":"audit_retention_days","before":90,"after":45}]`, event.DiffJSON)
}

func singleRecordedAuditEvent(t *testing.T, storage *fakeStorage) *AuditEvent {
	t.Helper()

	require.Len(t, storage.outbox, 1)

	data, ok := storage.outbox[0].Data.(*AuditEventRecordedData)
	require.True(t, ok, "outbox payload must be AuditEventRecordedData, got %T", storage.outbox[0].Data)
	require.NotNil(t, data.Event)

	return data.Event
}

func TestUpdateAuditRetention_AcceptsBounds(t *testing.T) {
	for _, days := range []int{1, testConfig.MaxRetentionDays} {
		storage := newFakeStorage()
		uc := NewUpdateAuditRetention(storage, testConfig, NewRecordAuditEvent(storage, nil))
		workspaceID := uuid.New()

		result, err := uc.Execute(context.Background(), UpdateAuditRetentionParams{WorkspaceID: workspaceID, RetentionDays: days})
		require.NoError(t, err, "days=%d", days)

		assert.Equal(t, AuditRetention{RetentionDays: days, MaxRetentionDays: testConfig.MaxRetentionDays}, result)
		require.Len(t, storage.retention, 1)
		assert.Equal(t, workspaceID, storage.retention[0].WorkspaceID)
		assert.Equal(t, days, storage.retention[0].RetentionDays)
	}
}

func TestUpdateAuditRetention_OverwritesExistingSetting(t *testing.T) {
	storage := newFakeStorage()
	uc := NewUpdateAuditRetention(storage, testConfig, NewRecordAuditEvent(storage, nil))
	workspaceID := uuid.New()

	_, err := uc.Execute(context.Background(), UpdateAuditRetentionParams{WorkspaceID: workspaceID, RetentionDays: 30})
	require.NoError(t, err)

	_, err = uc.Execute(context.Background(), UpdateAuditRetentionParams{WorkspaceID: workspaceID, RetentionDays: 60})
	require.NoError(t, err)

	require.Len(t, storage.retention, 1)
	assert.Equal(t, 60, storage.retention[0].RetentionDays)
}

func TestGetAuditRetention_DefaultFallback(t *testing.T) {
	uc := NewGetAuditRetention(newFakeStorage(), testConfig)

	result, err := uc.Execute(context.Background(), uuid.New())
	require.NoError(t, err)

	assert.Equal(t, AuditRetention{RetentionDays: 90, MaxRetentionDays: 365}, result)
}

func TestGetAuditRetention_StoredSetting(t *testing.T) {
	storage := newFakeStorage()
	workspaceID := uuid.New()
	storage.retention = []*AuditRetentionSetting{{WorkspaceID: workspaceID, RetentionDays: 30}}
	uc := NewGetAuditRetention(storage, testConfig)

	result, err := uc.Execute(context.Background(), workspaceID)
	require.NoError(t, err)

	assert.Equal(t, AuditRetention{RetentionDays: 30, MaxRetentionDays: 365}, result)
}

func TestGetAuditRetention_ClampsStoredSettingToMax(t *testing.T) {
	storage := newFakeStorage()
	workspaceID := uuid.New()
	storage.retention = []*AuditRetentionSetting{{WorkspaceID: workspaceID, RetentionDays: 400}}
	uc := NewGetAuditRetention(storage, testConfig)

	result, err := uc.Execute(context.Background(), workspaceID)
	require.NoError(t, err)

	assert.Equal(t, 365, result.RetentionDays)
}

func TestCleanupOldAuditEvents_UsesWorkspaceCutoffOrDefault(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	configured := uuid.New()
	defaulted := uuid.New()
	storage.retention = []*AuditRetentionSetting{{WorkspaceID: configured, RetentionDays: 10}}

	keepConfigured := addAuditEvent(storage, configured, now.Add(-9*24*time.Hour))
	addAuditEvent(storage, configured, now.Add(-11*24*time.Hour))
	keepDefault := addAuditEvent(storage, defaulted, now.Add(-11*24*time.Hour))
	keepDefaultEdge := addAuditEvent(storage, defaulted, now.Add(-89*24*time.Hour))
	addAuditEvent(storage, defaulted, now.Add(-91*24*time.Hour))

	uc := NewCleanupOldAuditEvents(storage, storage, testConfig, nil)
	uc.now = func() time.Time { return now }

	require.NoError(t, uc.Execute(context.Background()))

	remaining := make([]uuid.UUID, 0, len(storage.events))
	for _, event := range storage.events {
		remaining = append(remaining, event.ID)
	}

	assert.ElementsMatch(t, []uuid.UUID{keepConfigured, keepDefault, keepDefaultEdge}, remaining)
}

func addAuditEvent(storage *fakeStorage, workspaceID uuid.UUID, createdAt time.Time) uuid.UUID {
	event := &AuditEvent{
		ID:           uuid.New(),
		WorkspaceID:  workspaceID,
		ActorType:    ActorTypeSystem,
		Action:       ActionUpdate,
		ResourceType: ResourceTypeWorkspace,
		CreatedAt:    createdAt,
	}
	storage.events = append(storage.events, event)

	return event.ID
}
