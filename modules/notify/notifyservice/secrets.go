package notifyservice

import (
	"context"

	"github.com/google/uuid"
)

// SecretsProvider manages encrypted secret storage for the notify module.
type SecretsProvider interface {
	StoreSecret(ctx context.Context, ownerType string, ownerID uuid.UUID, plaintext string) (secretRef string, err error)
	RetrieveSecret(ctx context.Context, secretRef string) (plaintext string, err error)
	DeleteSecret(ctx context.Context, secretRef string) error
	DeleteByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error
}

// sensitiveChannelFields maps channel types to their sensitive config field paths.
// These fields are encrypted at rest via SecretsProvider.
var sensitiveChannelFields = map[ChannelType][]string{
	ChannelTypeSlack:    {"webhook_url"},
	ChannelTypeTelegram: {"bot_token"},
	// Email: no sensitive fields in config (SMTP creds are server-side env vars)
}

// IsSensitiveField returns true if the given field is sensitive for the channel type.
func IsSensitiveField(channelType ChannelType, field string) bool {
	for _, f := range sensitiveChannelFields[channelType] {
		if f == field {
			return true
		}
	}
	return false
}
