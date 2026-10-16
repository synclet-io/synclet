package notifyservice

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type purgeLog struct {
	entries   []string
	secretErr error
}

type purgeStorage struct {
	Storage
	log      *purgeLog
	channels []*NotificationChannel
	webhooks []*Webhook
}

func (s *purgeStorage) NotificationRules() NotificationRulesStorage {
	return &purgeRules{log: s.log}
}

func (s *purgeStorage) NotificationChannels() NotificationChannelsStorage {
	return &purgeChannels{log: s.log, records: s.channels}
}

func (s *purgeStorage) Webhooks() WebhooksStorage {
	return &purgeWebhooks{log: s.log, records: s.webhooks}
}

type purgeRules struct {
	NotificationRulesStorage
	log *purgeLog
}

func (r *purgeRules) Delete(_ context.Context, f *NotificationRuleFilter) error {
	r.log.entries = append(r.log.entries, "delete rules "+f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]).Value.String())

	return nil
}

type purgeChannels struct {
	NotificationChannelsStorage
	log     *purgeLog
	records []*NotificationChannel
}

func (c *purgeChannels) Find(context.Context, *NotificationChannelFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*NotificationChannel, error) {
	return c.records, nil
}

func (c *purgeChannels) Delete(context.Context, *NotificationChannelFilter) error {
	c.log.entries = append(c.log.entries, "delete channels")

	return nil
}

type purgeWebhooks struct {
	WebhooksStorage
	log     *purgeLog
	records []*Webhook
}

func (w *purgeWebhooks) Find(context.Context, *WebhookFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*Webhook, error) {
	return w.records, nil
}

func (w *purgeWebhooks) Delete(context.Context, *WebhookFilter) error {
	w.log.entries = append(w.log.entries, "delete webhooks")

	return nil
}

type purgeSecrets struct {
	SecretsProvider
	log *purgeLog
}

func (s *purgeSecrets) DeleteByOwner(_ context.Context, ownerType string, ownerID uuid.UUID) error {
	if s.log.secretErr != nil {
		return s.log.secretErr
	}

	s.log.entries = append(s.log.entries, "secrets "+ownerType+" "+ownerID.String())

	return nil
}

func TestPurgeWorkspace(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()

	t.Run("deletes rules, channels and webhooks with their secrets first", func(t *testing.T) {
		channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: workspaceID}
		webhook := &Webhook{ID: uuid.New(), WorkspaceID: workspaceID}
		log := &purgeLog{}
		storage := &purgeStorage{log: log, channels: []*NotificationChannel{channel}, webhooks: []*Webhook{webhook}}

		require.NoError(t, NewPurgeWorkspace(storage, &purgeSecrets{log: log}).Execute(ctx, workspaceID))

		assert.Equal(t, []string{
			"delete rules " + workspaceID.String(),
			"secrets channel " + channel.ID.String(),
			"delete channels",
			"secrets webhook " + webhook.ID.String(),
			"delete webhooks",
		}, log.entries)
	})

	t.Run("keeps channels when their secrets cannot be deleted", func(t *testing.T) {
		errBoom := errors.New("boom")
		log := &purgeLog{secretErr: errBoom}
		storage := &purgeStorage{log: log, channels: []*NotificationChannel{{ID: uuid.New(), WorkspaceID: workspaceID}}}

		require.ErrorIs(t, NewPurgeWorkspace(storage, &purgeSecrets{log: log}).Execute(ctx, workspaceID), errBoom)
		assert.NotContains(t, log.entries, "delete channels")
	})
}
