package notifyservice

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

const storedChannelRef = "$secret:00000000-0000-0000-0000-000000000001"

type recordingSecrets struct {
	SecretsProvider
	stored  map[string]string
	deleted []string
}

func (s *recordingSecrets) StoreSecret(_ context.Context, _ string, _ uuid.UUID, plaintext string) (string, error) {
	ref := secretutil.MakeSecretRef(uuid.New())
	s.stored[ref] = plaintext

	return ref, nil
}

func (s *recordingSecrets) DeleteSecret(_ context.Context, ref string) error {
	s.deleted = append(s.deleted, ref)

	return nil
}

func newUpdateChannelFixture(t *testing.T, channelType ChannelType, config map[string]string) (*UpdateChannel, *NotificationChannel, *recordingSecrets) {
	t.Helper()

	raw, err := json.Marshal(config)
	require.NoError(t, err)

	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: uuid.New(), Name: "ops", ChannelType: channelType, Config: string(raw)}
	secrets := &recordingSecrets{stored: map[string]string{}}

	return NewUpdateChannel(newTestStorage([]*NotificationChannel{channel}, nil), secrets, &capturingAuditRecorder{}), channel, secrets
}

func channelConfig(t *testing.T, channel *NotificationChannel) map[string]string {
	t.Helper()

	var config map[string]string
	require.NoError(t, json.Unmarshal([]byte(channel.Config), &config))

	return config
}

func TestUpdateChannel_MaskKeepsStoredSecret(t *testing.T) {
	uc, channel, secrets := newUpdateChannelFixture(t, ChannelTypeTelegram,
		map[string]string{"bot_token": storedChannelRef, "chat_id": "1"})

	updated, err := uc.Execute(context.Background(), UpdateChannelParams{
		ID:          channel.ID,
		WorkspaceID: channel.WorkspaceID,
		Config:      map[string]string{"bot_token": secretutil.SecretMask, "chat_id": "2"},
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"bot_token": storedChannelRef, "chat_id": "2"}, channelConfig(t, updated))
	assert.Empty(t, secrets.stored)
	assert.Empty(t, secrets.deleted)
}

func TestUpdateChannel_NewSecretReplacesOld(t *testing.T) {
	uc, channel, secrets := newUpdateChannelFixture(t, ChannelTypeSlack,
		map[string]string{"webhook_url": storedChannelRef})

	updated, err := uc.Execute(context.Background(), UpdateChannelParams{
		ID:          channel.ID,
		WorkspaceID: channel.WorkspaceID,
		Config:      map[string]string{"webhook_url": "https://hooks.slack.com/services/new"},
	})
	require.NoError(t, err)

	newRef := channelConfig(t, updated)["webhook_url"]
	assert.True(t, secretutil.IsSecretRef(newRef))
	assert.Equal(t, "https://hooks.slack.com/services/new", secrets.stored[newRef])
	assert.Equal(t, []string{storedChannelRef}, secrets.deleted)
}

func TestUpdateChannel_RejectsForeignSecretRef(t *testing.T) {
	uc, channel, secrets := newUpdateChannelFixture(t, ChannelTypeSlack,
		map[string]string{"webhook_url": storedChannelRef})

	_, err := uc.Execute(context.Background(), UpdateChannelParams{
		ID:          channel.ID,
		WorkspaceID: channel.WorkspaceID,
		Config:      map[string]string{"webhook_url": "$secret:00000000-0000-0000-0000-000000000002"},
	})

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Empty(t, secrets.deleted)
}

func TestUpdateChannel_MaskWithoutStoredSecretFailsValidation(t *testing.T) {
	uc, channel, secrets := newUpdateChannelFixture(t, ChannelTypeSlack, map[string]string{})

	_, err := uc.Execute(context.Background(), UpdateChannelParams{
		ID:          channel.ID,
		WorkspaceID: channel.WorkspaceID,
		Config:      map[string]string{"webhook_url": secretutil.SecretMask},
	})

	require.ErrorIs(t, err, ErrWebhookURLRequired)
	assert.Empty(t, secrets.stored)
}
