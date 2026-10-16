package notifyservice

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const storedWebhookRef = "$secret:00000000-0000-0000-0000-000000000002"

type webhookStorage struct {
	*testStorage
	webhooks *singleWebhookStore
}

func (s *webhookStorage) Webhooks() WebhooksStorage { return s.webhooks }

type singleWebhookStore struct {
	WebhooksStorage
	webhook *Webhook
}

func (s *singleWebhookStore) First(_ context.Context, _ *WebhookFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*Webhook, error) {
	cp := *s.webhook

	return &cp, nil
}

func (s *singleWebhookStore) Update(_ context.Context, webhook *Webhook) (*Webhook, error) {
	cp := *webhook
	s.webhook = &cp

	return webhook, nil
}

func newUpdateWebhookFixture() (*UpdateWebhook, *singleWebhookStore, *recordingSecrets, *capturingAuditRecorder) {
	store := &singleWebhookStore{webhook: &Webhook{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		URL:         "https://example.com/hook",
		Events:      `["sync.completed"]`,
		Secret:      storedWebhookRef,
		Enabled:     true,
	}}
	secrets := &recordingSecrets{stored: map[string]string{}}
	audit := &capturingAuditRecorder{}

	return NewUpdateWebhook(&webhookStorage{testStorage: newTestStorage(nil, nil), webhooks: store}, secrets, audit), store, secrets, audit
}

func TestUpdateWebhook_RotatesSecret(t *testing.T) {
	uc, store, secrets, audit := newUpdateWebhookFixture()
	secret := "new-signing-secret" //nolint:gosec // test fixture

	_, err := uc.Execute(context.Background(), UpdateWebhookParams{ID: store.webhook.ID, WorkspaceID: store.webhook.WorkspaceID, Secret: &secret})
	require.NoError(t, err)

	require.Len(t, secrets.stored, 1)
	assert.Equal(t, secret, secrets.stored[store.webhook.Secret])
	assert.Equal(t, []string{storedWebhookRef}, secrets.deleted)
	require.Len(t, audit.events, 1)
	assert.NotEqual(t, audit.events[0].Before.(map[string]any)["secret"], audit.events[0].After.(map[string]any)["secret"])
}

func TestUpdateWebhook_EmptySecretStopsSigning(t *testing.T) {
	uc, store, secrets, _ := newUpdateWebhookFixture()
	empty := ""

	_, err := uc.Execute(context.Background(), UpdateWebhookParams{ID: store.webhook.ID, WorkspaceID: store.webhook.WorkspaceID, Secret: &empty})
	require.NoError(t, err)

	assert.Empty(t, store.webhook.Secret)
	assert.Empty(t, secrets.stored)
	assert.Equal(t, []string{storedWebhookRef}, secrets.deleted)
}

func TestUpdateWebhook_Events(t *testing.T) {
	t.Run("nil keeps events", func(t *testing.T) {
		uc, store, secrets, _ := newUpdateWebhookFixture()

		_, err := uc.Execute(context.Background(), UpdateWebhookParams{ID: store.webhook.ID, WorkspaceID: store.webhook.WorkspaceID})
		require.NoError(t, err)

		assert.JSONEq(t, `["sync.completed"]`, store.webhook.Events)
		assert.Equal(t, storedWebhookRef, store.webhook.Secret)
		assert.Empty(t, secrets.deleted)
	})

	t.Run("empty clears events", func(t *testing.T) {
		uc, store, _, _ := newUpdateWebhookFixture()

		_, err := uc.Execute(context.Background(), UpdateWebhookParams{ID: store.webhook.ID, WorkspaceID: store.webhook.WorkspaceID, Events: []string{}})
		require.NoError(t, err)

		assert.JSONEq(t, `[]`, store.webhook.Events)
	})
}
