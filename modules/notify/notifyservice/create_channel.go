package notifyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// CreateChannelParams holds parameters for creating a notification channel.
type CreateChannelParams struct {
	WorkspaceID uuid.UUID
	Name        string
	ChannelType ChannelType
	Config      map[string]string
	Enabled     bool
}

// CreateChannel creates a new notification channel.
type CreateChannel struct {
	storage Storage
	secrets SecretsProvider
}

// NewCreateChannel creates a new CreateChannel use case.
func NewCreateChannel(storage Storage, secrets SecretsProvider) *CreateChannel {
	return &CreateChannel{storage: storage, secrets: secrets}
}

// Execute creates a notification channel with the given parameters.
func (uc *CreateChannel) Execute(ctx context.Context, params CreateChannelParams) (*NotificationChannel, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	if !params.ChannelType.IsValid() {
		return nil, fmt.Errorf("invalid channel_type: must be one of slack, email, telegram")
	}

	if err := validateChannelConfig(params.ChannelType, params.Config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	channelID := uuid.New()

	// Encrypt sensitive config fields, tracking stored refs for cleanup on partial failure.
	var storedRefs []string
	for field, value := range params.Config {
		if IsSensitiveField(params.ChannelType, field) && value != "" {
			ref, err := uc.secrets.StoreSecret(ctx, "channel", channelID, value)
			if err != nil {
				// Clean up already-stored secrets on failure (best-effort).
				for _, r := range storedRefs {
					if delErr := uc.secrets.DeleteSecret(ctx, r); delErr != nil {
						slog.Error("failed to clean up orphaned secret", "ref", r, "error", delErr)
					}
				}
				return nil, fmt.Errorf("encrypting channel config field %s: %w", field, err)
			}
			storedRefs = append(storedRefs, ref)
			params.Config[field] = ref
		}
	}

	configJSON, err := json.Marshal(params.Config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	now := time.Now()
	channel := &NotificationChannel{
		ID:          channelID,
		WorkspaceID: params.WorkspaceID,
		Name:        params.Name,
		ChannelType: params.ChannelType,
		Config:      string(configJSON),
		Enabled:     params.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := uc.storage.NotificationChannels().Create(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("creating notification channel: %w", err)
	}

	return created, nil
}

func validateChannelConfig(channelType ChannelType, config map[string]string) error {
	switch channelType {
	case ChannelTypeSlack:
		if config["webhook_url"] == "" {
			return fmt.Errorf("webhook_url is required for slack channels")
		}
	case ChannelTypeEmail:
		if config["recipients"] == "" {
			return fmt.Errorf("recipients is required for email channels")
		}
	case ChannelTypeTelegram:
		if config["bot_token"] == "" {
			return fmt.Errorf("bot_token is required for telegram channels")
		}
		if config["chat_id"] == "" {
			return fmt.Errorf("chat_id is required for telegram channels")
		}
	}

	return nil
}
