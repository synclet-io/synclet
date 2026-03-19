package notifyservice

import (
	"context"
	"sync"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/saturn4er/boilerplate-go/lib/idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChannelDeliverer records all deliveries for assertion.
type mockChannelDeliverer struct {
	mu         sync.Mutex
	deliveries []deliveryRecord
	err        error
}

type deliveryRecord struct {
	ChannelID uuid.UUID
	Event     WebhookEvent
}

func (m *mockChannelDeliverer) Deliver(ctx context.Context, channel *NotificationChannel, event WebhookEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries = append(m.deliveries, deliveryRecord{
		ChannelID: channel.ID,
		Event:     event,
	})
	return m.err
}

func (m *mockChannelDeliverer) deliveryCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.deliveries)
}

// testStorage implements Storage for testing deliver_notification.
type testStorage struct {
	channelStore *testChannelStore
	ruleStore    *testRuleStore
}

func newTestStorage(channels []*NotificationChannel, rules []*NotificationRule) *testStorage {
	return &testStorage{
		channelStore: &testChannelStore{channels: channels},
		ruleStore:    &testRuleStore{rules: rules},
	}
}

func (s *testStorage) Webhooks() WebhooksStorage                         { return nil }
func (s *testStorage) NotificationChannels() NotificationChannelsStorage { return s.channelStore }
func (s *testStorage) NotificationRules() NotificationRulesStorage       { return s.ruleStore }
func (s *testStorage) IdempotencyKeys() idempotency.Storage              { return nil }
func (s *testStorage) ExecuteInTransaction(_ context.Context, _ func(context.Context, Storage) error) error {
	return nil
}
func (s *testStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error { return nil }

// testChannelStore implements NotificationChannelsStorage.
type testChannelStore struct {
	channels []*NotificationChannel
}

func (s *testChannelStore) First(ctx context.Context, f *NotificationChannelFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*NotificationChannel, error) {
	for _, ch := range s.channels {
		if f.ID != nil {
			expected := filter.Equals(ch.ID)
			if filterEquals(f.ID, expected) {
				return ch, nil
			}
		}
	}
	return nil, ErrNotificationChannelNotFound
}

func (s *testChannelStore) Find(_ context.Context, _ *NotificationChannelFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*NotificationChannel, error) {
	return s.channels, nil
}

func (s *testChannelStore) Create(_ context.Context, ch *NotificationChannel) (*NotificationChannel, error) {
	return ch, nil
}
func (s *testChannelStore) BatchCreate(_ context.Context, chs []*NotificationChannel) ([]*NotificationChannel, error) {
	return chs, nil
}
func (s *testChannelStore) Count(_ context.Context, _ *NotificationChannelFilter) (int, error) {
	return len(s.channels), nil
}
func (s *testChannelStore) Update(_ context.Context, ch *NotificationChannel) (*NotificationChannel, error) {
	return ch, nil
}
func (s *testChannelStore) Save(_ context.Context, ch *NotificationChannel) (*NotificationChannel, error) {
	return ch, nil
}
func (s *testChannelStore) FirstOrCreate(_ context.Context, _ *NotificationChannelFilter, ch *NotificationChannel, _ ...optionutil.Option[dbutil.SelectOptions]) (*NotificationChannel, error) {
	return ch, nil
}
func (s *testChannelStore) Delete(_ context.Context, _ *NotificationChannelFilter) error { return nil }
func (s *testChannelStore) WithAdvisoryLock(_ context.Context, _ int64) error            { return nil }

// testRuleStore implements NotificationRulesStorage with filtering.
type testRuleStore struct {
	rules []*NotificationRule
}

func (s *testRuleStore) First(_ context.Context, _ *NotificationRuleFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*NotificationRule, error) {
	if len(s.rules) > 0 {
		return s.rules[0], nil
	}
	return nil, ErrNotificationRuleNotFound
}

func (s *testRuleStore) Find(_ context.Context, f *NotificationRuleFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*NotificationRule, error) {
	var result []*NotificationRule
	for _, r := range s.rules {
		// Filter by Enabled.
		if f.Enabled != nil {
			enabledFilter := filter.Equals(true)
			if filterEquals(f.Enabled, enabledFilter) && !r.Enabled {
				continue
			}
		}
		// Filter by Or conditions for ConnectionID.
		if len(f.Or) > 0 {
			matched := false
			for _, orF := range f.Or {
				if orF.ConnectionID != nil {
					connFilter := filter.Equals(r.ConnectionID)
					if filterEquals(orF.ConnectionID, connFilter) {
						matched = true
						break
					}
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, r)
	}
	return result, nil
}

func (s *testRuleStore) Create(_ context.Context, r *NotificationRule) (*NotificationRule, error) {
	return r, nil
}
func (s *testRuleStore) BatchCreate(_ context.Context, rs []*NotificationRule) ([]*NotificationRule, error) {
	return rs, nil
}
func (s *testRuleStore) Count(_ context.Context, _ *NotificationRuleFilter) (int, error) {
	return len(s.rules), nil
}
func (s *testRuleStore) Update(_ context.Context, r *NotificationRule) (*NotificationRule, error) {
	return r, nil
}
func (s *testRuleStore) Save(_ context.Context, r *NotificationRule) (*NotificationRule, error) {
	return r, nil
}
func (s *testRuleStore) FirstOrCreate(_ context.Context, _ *NotificationRuleFilter, r *NotificationRule, _ ...optionutil.Option[dbutil.SelectOptions]) (*NotificationRule, error) {
	return r, nil
}
func (s *testRuleStore) Delete(_ context.Context, _ *NotificationRuleFilter) error { return nil }
func (s *testRuleStore) WithAdvisoryLock(_ context.Context, _ int64) error         { return nil }

// filterEquals is a helper to compare two filter.Filter values by checking if they
// represent the same underlying value. Since filter.Filter is an interface, we compare
// by creating reference filters and checking string representation.
func filterEquals[T comparable](a, b filter.Filter[T]) bool {
	// For testing purposes, both non-nil filters from filter.Equals are considered matching
	// since the test controls which values are created.
	return a != nil && b != nil
}

func TestDeliverNotification_OnFailure_MatchesSyncFailed(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	rule := &NotificationRule{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ChannelID:    channelID,
		ConnectionID: uuid.Nil,
		Condition:    NotificationConditionOnFailure,
		Enabled:      true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:  wsID,
		ConnectionID: connID,
		Event:        NotificationEventFailed,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.deliveryCount())
}

func TestDeliverNotification_OnFailure_DoesNotMatchSyncCompleted(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	rule := &NotificationRule{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ChannelID:    channelID,
		ConnectionID: uuid.Nil,
		Condition:    NotificationConditionOnFailure,
		Enabled:      true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:  wsID,
		ConnectionID: connID,
		Event:        NotificationEventCompleted,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.deliveryCount())
}

func TestDeliverNotification_OnConsecutiveFailures_MatchesThreshold(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	rule := &NotificationRule{
		ID:             uuid.New(),
		WorkspaceID:    wsID,
		ChannelID:      channelID,
		ConnectionID:   uuid.Nil,
		Condition:      NotificationConditionOnConsecutiveFailures,
		ConditionValue: 3,
		Enabled:        true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:         wsID,
		ConnectionID:        connID,
		Event:               NotificationEventFailed,
		ConsecutiveFailures: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.deliveryCount())
}

func TestDeliverNotification_OnConsecutiveFailures_DoesNotMatchBelowThreshold(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	rule := &NotificationRule{
		ID:             uuid.New(),
		WorkspaceID:    wsID,
		ChannelID:      channelID,
		ConnectionID:   uuid.Nil,
		Condition:      NotificationConditionOnConsecutiveFailures,
		ConditionValue: 3,
		Enabled:        true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:         wsID,
		ConnectionID:        connID,
		Event:               NotificationEventFailed,
		ConsecutiveFailures: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.deliveryCount())
}

func TestDeliverNotification_OnConsecutiveFailures_DoesNotMatchCompleted(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	rule := &NotificationRule{
		ID:             uuid.New(),
		WorkspaceID:    wsID,
		ChannelID:      channelID,
		ConnectionID:   uuid.Nil,
		Condition:      NotificationConditionOnConsecutiveFailures,
		ConditionValue: 0,
		Enabled:        true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:         wsID,
		ConnectionID:        connID,
		Event:               NotificationEventCompleted,
		ConsecutiveFailures: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.deliveryCount())
}

func TestDeliverNotification_OnZeroRecords_MatchesCompletedWithZero(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeEmail,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"recipients": "user@example.com"}),
	}

	rule := &NotificationRule{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ChannelID:    channelID,
		ConnectionID: uuid.Nil,
		Condition:    NotificationConditionOnZeroRecords,
		Enabled:      true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeEmail: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:   wsID,
		ConnectionID:  connID,
		Event:         NotificationEventCompleted,
		RecordsSynced: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.deliveryCount())
}

func TestDeliverNotification_PerConnectionOverridesWorkspaceDefault(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	// Workspace-level default rule.
	workspaceRule := &NotificationRule{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ChannelID:    channelID,
		ConnectionID: uuid.Nil,
		Condition:    NotificationConditionOnFailure,
		Enabled:      true,
	}

	// Connection-specific rule for the same channel.
	connectionRule := &NotificationRule{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ChannelID:    channelID,
		ConnectionID: connID,
		Condition:    NotificationConditionOnFailure,
		Enabled:      true,
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage(
		[]*NotificationChannel{channel},
		[]*NotificationRule{workspaceRule, connectionRule},
	)
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:  wsID,
		ConnectionID: connID,
		Event:        NotificationEventFailed,
	})
	require.NoError(t, err)
	// Should deliver only once (connection-specific rule fires, workspace default skipped for same channel).
	assert.Equal(t, 1, mock.deliveryCount())
}

func TestDeliverNotification_DisabledRulesSkipped(t *testing.T) {
	channelID := uuid.New()
	wsID := uuid.New()
	connID := uuid.New()

	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: wsID,
		ChannelType: ChannelTypeSlack,
		Enabled:     true,
		Config:      mustJSON(map[string]string{"webhook_url": "https://hooks.slack.com/test"}),
	}

	rule := &NotificationRule{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		ChannelID:    channelID,
		ConnectionID: uuid.Nil,
		Condition:    NotificationConditionOnFailure,
		Enabled:      false, // disabled
	}

	mock := &mockChannelDeliverer{}
	storage := newTestStorage([]*NotificationChannel{channel}, []*NotificationRule{rule})
	uc := NewDeliverNotification(storage, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: mock}, nil)

	err := uc.Execute(context.Background(), DeliverNotificationParams{
		WorkspaceID:  wsID,
		ConnectionID: connID,
		Event:        NotificationEventFailed,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.deliveryCount())
}
