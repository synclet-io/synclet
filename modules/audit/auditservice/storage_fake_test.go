package auditservice

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
)

// fakeStorage is the shared in-memory Storage and RetentionStorage for audit
// use case tests.
type fakeStorage struct {
	mu        sync.Mutex
	events    []*AuditEvent
	retention []*AuditRetentionSetting
	outbox    []*AuditOutboxEvent
	outboxErr error
	createErr error
}

var (
	_ Storage          = (*fakeStorage)(nil)
	_ RetentionStorage = (*fakeStorage)(nil)
	_ EventPageStorage = (*fakeStorage)(nil)
)

func newFakeStorage() *fakeStorage {
	return &fakeStorage{}
}

func (s *fakeStorage) AuditEvents() AuditEventsStorage {
	return &fakeAuditEventsStorage{parent: s}
}

func (s *fakeStorage) AuditRetentionSettings() AuditRetentionSettingsStorage {
	return &fakeRetentionSettingsStorage{parent: s}
}

func (s *fakeStorage) AuditOutboxEvents() AuditOutboxEventsOutbox {
	return &fakeOutbox{parent: s}
}

// fakeOutbox records every event sent via the outbox so tests can assert on
// emitted events, or fails with outboxErr when set.
type fakeOutbox struct {
	parent *fakeStorage
}

func (o *fakeOutbox) Send(_ context.Context, event *AuditOutboxEvent) error {
	o.parent.mu.Lock()
	defer o.parent.mu.Unlock()

	if o.parent.outboxErr != nil {
		return o.parent.outboxErr
	}

	o.parent.outbox = append(o.parent.outbox, event)

	return nil
}

func (s *fakeStorage) IdempotencyKeys() idempotency.Storage {
	panic("IdempotencyKeys not implemented")
}

func (s *fakeStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error {
	return cb(ctx, s)
}

func (s *fakeStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error {
	return nil
}

func (s *fakeStorage) DistinctWorkspaceIDs(_ context.Context) ([]uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := map[uuid.UUID]struct{}{}
	ids := []uuid.UUID{}

	for _, event := range s.events {
		if _, ok := seen[event.WorkspaceID]; ok {
			continue
		}

		seen[event.WorkspaceID] = struct{}{}
		ids = append(ids, event.WorkspaceID)
	}

	return ids, nil
}

func (s *fakeStorage) DeleteOlderThan(_ context.Context, workspaceID uuid.UUID, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64

	kept := s.events[:0]
	for _, event := range s.events {
		if event.WorkspaceID == workspaceID && event.CreatedAt.Before(cutoff) {
			deleted++

			continue
		}

		kept = append(kept, event)
	}
	s.events = kept

	return deleted, nil
}

func equalsValue[T comparable](flt filter.Filter[T]) (T, bool) {
	var zero T

	if flt == nil {
		return zero, false
	}

	if eq, ok := flt.(*filter.EqualsFilter[T]); ok {
		return eq.Value, true
	}

	return zero, false
}

type fakeAuditEventsStorage struct {
	parent *fakeStorage
}

func (s *fakeAuditEventsStorage) matches(event *AuditEvent, flt *AuditEventFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.WorkspaceID); ok && event.WorkspaceID != id {
		return false
	}

	if id, ok := equalsValue[uuid.UUID](flt.ActorID); ok && event.ActorID != id {
		return false
	}

	if actorType, ok := equalsValue[ActorType](flt.ActorType); ok && event.ActorType != actorType {
		return false
	}

	if !matchesTime(flt.CreatedAt, event.CreatedAt) {
		return false
	}

	for _, sub := range flt.And {
		if !s.matches(event, sub) {
			return false
		}
	}

	return true
}

func matchesTime(flt filter.Filter[time.Time], value time.Time) bool {
	switch typed := flt.(type) {
	case nil:
		return true
	case *filter.GreaterOrEqualsFilter[time.Time]:
		return !value.Before(typed.Value)
	case *filter.LessFilter[time.Time]:
		return value.Before(typed.Value)
	case *filter.AndFilter[time.Time]:
		for _, sub := range typed.Filters {
			if !matchesTime(sub, value) {
				return false
			}
		}

		return true
	default:
		panic(fmt.Sprintf("unsupported created_at filter %T", flt))
	}
}

func (s *fakeAuditEventsStorage) Create(_ context.Context, event *AuditEvent) (*AuditEvent, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	if s.parent.createErr != nil {
		return nil, s.parent.createErr
	}

	for _, existing := range s.parent.events {
		if existing.ID == event.ID {
			return nil, ErrAuditEventAlreadyExists
		}
	}

	s.parent.events = append(s.parent.events, event)

	return event, nil
}

func (s *fakeAuditEventsStorage) BatchCreate(_ context.Context, _ []*AuditEvent) ([]*AuditEvent, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeAuditEventsStorage) Count(_ context.Context, _ *AuditEventFilter) (int, error) {
	panic("Count not implemented")
}

func (s *fakeAuditEventsStorage) Update(_ context.Context, event *AuditEvent) (*AuditEvent, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.events {
		if existing.ID == event.ID {
			updated := event.Copy()
			s.parent.events[i] = &updated

			return &updated, nil
		}
	}

	return nil, ErrAuditEventNotFound
}

func (s *fakeAuditEventsStorage) Save(_ context.Context, _ *AuditEvent) (*AuditEvent, error) {
	panic("Save not implemented")
}

func (s *fakeAuditEventsStorage) First(_ context.Context, _ *AuditEventFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*AuditEvent, error) {
	panic("First not implemented")
}

func (s *fakeAuditEventsStorage) FirstOrCreate(_ context.Context, _ *AuditEventFilter, _ *AuditEvent, _ ...optionutil.Option[dbutil.SelectOptions]) (*AuditEvent, error) {
	panic("FirstOrCreate not implemented")
}

func (s *fakeAuditEventsStorage) Find(_ context.Context, flt *AuditEventFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*AuditEvent, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	out := []*AuditEvent{}
	for _, event := range s.parent.events {
		if s.matches(event, flt) {
			out = append(out, event)
		}
	}

	return out, nil
}

func (s *fakeStorage) FindPage(ctx context.Context, flt *AuditEventFilter, offset, limit int) ([]*AuditEvent, error) {
	events, err := s.AuditEvents().Find(ctx, flt)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(events, func(a, b *AuditEvent) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})

	if offset >= len(events) {
		return []*AuditEvent{}, nil
	}

	return events[offset:min(offset+limit, len(events))], nil
}

func (s *fakeAuditEventsStorage) Delete(_ context.Context, _ *AuditEventFilter) error {
	panic("Delete not implemented")
}

func (s *fakeAuditEventsStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	return nil
}

type fakeRetentionSettingsStorage struct {
	parent *fakeStorage
}

func (s *fakeRetentionSettingsStorage) matches(setting *AuditRetentionSetting, flt *AuditRetentionSettingFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.WorkspaceID); ok && setting.WorkspaceID != id {
		return false
	}

	return true
}

func (s *fakeRetentionSettingsStorage) Create(_ context.Context, setting *AuditRetentionSetting) (*AuditRetentionSetting, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, existing := range s.parent.retention {
		if existing.WorkspaceID == setting.WorkspaceID {
			return nil, ErrAuditRetentionSettingAlreadyExists
		}
	}

	s.parent.retention = append(s.parent.retention, setting)

	return setting, nil
}

func (s *fakeRetentionSettingsStorage) BatchCreate(_ context.Context, _ []*AuditRetentionSetting) ([]*AuditRetentionSetting, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeRetentionSettingsStorage) Count(_ context.Context, _ *AuditRetentionSettingFilter) (int, error) {
	panic("Count not implemented")
}

func (s *fakeRetentionSettingsStorage) Update(_ context.Context, setting *AuditRetentionSetting) (*AuditRetentionSetting, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.retention {
		if existing.WorkspaceID == setting.WorkspaceID {
			s.parent.retention[i] = setting

			return setting, nil
		}
	}

	return nil, ErrAuditRetentionSettingNotFound
}

func (s *fakeRetentionSettingsStorage) Save(ctx context.Context, setting *AuditRetentionSetting) (*AuditRetentionSetting, error) {
	if _, err := s.Update(ctx, setting); err == nil {
		return setting, nil
	}

	return s.Create(ctx, setting)
}

func (s *fakeRetentionSettingsStorage) First(_ context.Context, flt *AuditRetentionSettingFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*AuditRetentionSetting, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, setting := range s.parent.retention {
		if s.matches(setting, flt) {
			return setting, nil
		}
	}

	return nil, ErrAuditRetentionSettingNotFound
}

func (s *fakeRetentionSettingsStorage) FirstOrCreate(_ context.Context, _ *AuditRetentionSettingFilter, _ *AuditRetentionSetting, _ ...optionutil.Option[dbutil.SelectOptions]) (*AuditRetentionSetting, error) {
	panic("FirstOrCreate not implemented")
}

func (s *fakeRetentionSettingsStorage) Find(_ context.Context, flt *AuditRetentionSettingFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*AuditRetentionSetting, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	out := []*AuditRetentionSetting{}
	for _, setting := range s.parent.retention {
		if s.matches(setting, flt) {
			out = append(out, setting)
		}
	}

	return out, nil
}

func (s *fakeRetentionSettingsStorage) Delete(_ context.Context, _ *AuditRetentionSettingFilter) error {
	panic("Delete not implemented")
}

func (s *fakeRetentionSettingsStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	return nil
}
