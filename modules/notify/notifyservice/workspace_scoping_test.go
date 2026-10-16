package notifyservice

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopedMatch(f filter.Filter[uuid.UUID], value uuid.UUID) bool {
	return f == nil || f.(*filter.EqualsFilter[uuid.UUID]).Value == value
}

type scopedStorage struct {
	Storage
	channels *scopedChannels
	rules    *scopedRules
}

func (s *scopedStorage) NotificationChannels() NotificationChannelsStorage { return s.channels }
func (s *scopedStorage) NotificationRules() NotificationRulesStorage       { return s.rules }

type scopedChannels struct {
	NotificationChannelsStorage
	records []*NotificationChannel
	deleted int
}

func (s *scopedChannels) First(_ context.Context, f *NotificationChannelFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*NotificationChannel, error) {
	for _, rec := range s.records {
		if scopedMatch(f.ID, rec.ID) && scopedMatch(f.WorkspaceID, rec.WorkspaceID) {
			return rec, nil
		}
	}

	return nil, ErrNotificationChannelNotFound
}

func (s *scopedChannels) Delete(context.Context, *NotificationChannelFilter) error {
	s.deleted++

	return nil
}

type scopedRules struct {
	NotificationRulesStorage
	records []*NotificationRule
	deleted int
}

func (s *scopedRules) First(_ context.Context, f *NotificationRuleFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*NotificationRule, error) {
	for _, rec := range s.records {
		if scopedMatch(f.ID, rec.ID) && scopedMatch(f.WorkspaceID, rec.WorkspaceID) {
			return rec, nil
		}
	}

	return nil, ErrNotificationRuleNotFound
}

func (s *scopedRules) Create(_ context.Context, rec *NotificationRule) (*NotificationRule, error) {
	s.records = append(s.records, rec)

	return rec, nil
}

func (s *scopedRules) Delete(context.Context, *NotificationRuleFilter) error {
	s.deleted++

	return nil
}

type scopedSecrets struct {
	SecretsProvider
	deletedOwners []uuid.UUID
}

func (s *scopedSecrets) DeleteByOwner(_ context.Context, _ string, ownerID uuid.UUID) error {
	s.deletedOwners = append(s.deletedOwners, ownerID)

	return nil
}

type fakeConnectionChecker struct {
	workspaceID  uuid.UUID
	connectionID uuid.UUID
}

func (c fakeConnectionChecker) ConnectionExists(_ context.Context, workspaceID, connectionID uuid.UUID) (bool, error) {
	return workspaceID == c.workspaceID && connectionID == c.connectionID, nil
}

func newScopedStorage() *scopedStorage {
	return &scopedStorage{channels: &scopedChannels{}, rules: &scopedRules{}}
}

func TestCreateNotificationRule_WorkspaceScoping(t *testing.T) {
	ws := uuid.New()
	ownChannel := &NotificationChannel{ID: uuid.New(), WorkspaceID: ws}
	foreignChannel := &NotificationChannel{ID: uuid.New(), WorkspaceID: uuid.New()}
	ownConnection := uuid.New()

	tests := []struct {
		name         string
		channelID    uuid.UUID
		connectionID uuid.UUID
		wantErr      error
	}{
		{name: "own channel, workspace default", channelID: ownChannel.ID},
		{name: "own channel and connection", channelID: ownChannel.ID, connectionID: ownConnection},
		{name: "channel from another workspace", channelID: foreignChannel.ID, wantErr: ErrNotificationChannelNotFound},
		{name: "connection from another workspace", channelID: ownChannel.ID, connectionID: uuid.New(), wantErr: ErrConnectionNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newScopedStorage()
			store.channels.records = []*NotificationChannel{ownChannel, foreignChannel}
			recorder := &capturingAuditRecorder{}

			uc := NewCreateNotificationRule(store, fakeConnectionChecker{workspaceID: ws, connectionID: ownConnection}, recorder)
			_, err := uc.Execute(context.Background(), CreateNotificationRuleParams{
				WorkspaceID:  ws,
				ChannelID:    tt.channelID,
				ConnectionID: tt.connectionID,
				Condition:    NotificationConditionOnFailure,
				Enabled:      true,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, store.rules.records)
				assert.Empty(t, recorder.events)

				return
			}

			require.NoError(t, err)
			assert.Len(t, store.rules.records, 1)
			assert.Len(t, recorder.events, 1)
		})
	}
}

func TestDeleteNotificationRule_UnknownIDIsNotFound(t *testing.T) {
	ws := uuid.New()
	store := newScopedStorage()
	store.rules.records = []*NotificationRule{{ID: uuid.New(), WorkspaceID: uuid.New()}}
	recorder := &capturingAuditRecorder{}

	err := NewDeleteNotificationRule(store, recorder).Execute(context.Background(), DeleteNotificationRuleParams{
		ID:          store.rules.records[0].ID,
		WorkspaceID: ws,
	})
	require.ErrorIs(t, err, ErrNotificationRuleNotFound)
	assert.Zero(t, store.rules.deleted)
	assert.Empty(t, recorder.events)
}

func TestDeleteNotificationRule_DeletesAndAudits(t *testing.T) {
	ws := uuid.New()
	rule := &NotificationRule{ID: uuid.New(), WorkspaceID: ws, ChannelID: uuid.New(), Condition: NotificationConditionOnFailure}
	store := newScopedStorage()
	store.rules.records = []*NotificationRule{rule}
	recorder := &capturingAuditRecorder{}

	require.NoError(t, NewDeleteNotificationRule(store, recorder).Execute(context.Background(), DeleteNotificationRuleParams{
		ID:          rule.ID,
		WorkspaceID: ws,
	}))
	assert.Equal(t, 1, store.rules.deleted)
	require.Len(t, recorder.events, 1)
	assert.Equal(t, rule.ID, recorder.events[0].ResourceID)
}

func TestDeleteChannel_UnknownIDIsNotFound(t *testing.T) {
	store := newScopedStorage()
	store.channels.records = []*NotificationChannel{{ID: uuid.New(), WorkspaceID: uuid.New()}}
	secrets := &scopedSecrets{}
	recorder := &capturingAuditRecorder{}

	err := NewDeleteChannel(store, secrets, recorder).Execute(context.Background(), DeleteChannelParams{
		ID:          store.channels.records[0].ID,
		WorkspaceID: uuid.New(),
	})
	require.ErrorIs(t, err, ErrNotificationChannelNotFound)
	assert.Zero(t, store.channels.deleted)
	assert.Zero(t, store.rules.deleted)
	assert.Empty(t, secrets.deletedOwners)
	assert.Empty(t, recorder.events)
}

func TestDeleteChannel_DeletesRulesSecretsAndAudits(t *testing.T) {
	ws := uuid.New()
	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: ws, Name: "ops", ChannelType: ChannelTypeSlack}
	store := newScopedStorage()
	store.channels.records = []*NotificationChannel{channel}
	secrets := &scopedSecrets{}
	recorder := &capturingAuditRecorder{}

	require.NoError(t, NewDeleteChannel(store, secrets, recorder).Execute(context.Background(), DeleteChannelParams{
		ID:          channel.ID,
		WorkspaceID: ws,
	}))
	assert.Equal(t, 1, store.channels.deleted)
	assert.Equal(t, 1, store.rules.deleted)
	assert.Equal(t, []uuid.UUID{channel.ID}, secrets.deletedOwners)
	require.Len(t, recorder.events, 1)
	assert.Equal(t, "ops", recorder.events[0].ResourceLabel)
}
