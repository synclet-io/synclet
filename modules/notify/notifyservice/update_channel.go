package notifyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// UpdateChannelParams holds parameters for updating a notification channel.
type UpdateChannelParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        *string
	Config      map[string]string
	Enabled     *bool
}

// UpdateChannel updates an existing notification channel.
type UpdateChannel struct {
	storage Storage
	secrets SecretsProvider
}

// NewUpdateChannel creates a new UpdateChannel use case.
func NewUpdateChannel(storage Storage, secrets SecretsProvider) *UpdateChannel {
	return &UpdateChannel{storage: storage, secrets: secrets}
}

// Execute updates the notification channel matching the given ID and workspace.
func (uc *UpdateChannel) Execute(ctx context.Context, params UpdateChannelParams) (*NotificationChannel, error) {
	channel, err := uc.storage.NotificationChannels().First(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting notification channel: %w", err)
	}

	if params.Name != nil {
		channel.Name = *params.Name
	}

	if params.Config != nil {
		if err := validateChannelConfig(channel.ChannelType, params.Config); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		// Load existing config to find old secret refs for cleanup.
		var existingConfig map[string]string
		if err := json.Unmarshal([]byte(channel.Config), &existingConfig); err != nil {
			existingConfig = map[string]string{}
		}

		// Encrypt sensitive config fields.
		for field, value := range params.Config {
			if IsSensitiveField(channel.ChannelType, field) && value != "" && !secretutil.IsSecretRef(value) {
				// Delete old secret ref if it exists.
				if oldRef, ok := existingConfig[field]; ok && secretutil.IsSecretRef(oldRef) {
					_ = uc.secrets.DeleteSecret(ctx, oldRef) // non-fatal
				}

				ref, err := uc.secrets.StoreSecret(ctx, "channel", channel.ID, value)
				if err != nil {
					return nil, fmt.Errorf("encrypting channel config field %s: %w", field, err)
				}

				params.Config[field] = ref
			}
		}

		configJSON, err := json.Marshal(params.Config)
		if err != nil {
			return nil, fmt.Errorf("marshaling config: %w", err)
		}

		channel.Config = string(configJSON)
	}

	if params.Enabled != nil {
		channel.Enabled = *params.Enabled
	}

	channel.UpdatedAt = time.Now()

	updated, err := uc.storage.NotificationChannels().Update(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("updating notification channel: %w", err)
	}

	return updated, nil
}
