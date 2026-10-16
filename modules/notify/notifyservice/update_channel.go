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
	audit   AuditRecorder
}

// NewUpdateChannel creates a new UpdateChannel use case.
func NewUpdateChannel(storage Storage, secrets SecretsProvider, audit AuditRecorder) *UpdateChannel {
	return &UpdateChannel{storage: storage, secrets: secrets, audit: audit}
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

	before := map[string]any{
		"name":    channel.Name,
		"enabled": channel.Enabled,
	}

	if params.Name != nil {
		channel.Name = *params.Name
	}

	var staleRefs []string

	if params.Config != nil {
		var existingConfig map[string]string
		if err := json.Unmarshal([]byte(channel.Config), &existingConfig); err != nil {
			existingConfig = map[string]string{}
		}

		config, err := resolveMaskedChannelSecrets(channel.ChannelType, existingConfig, params.Config)
		if err != nil {
			return nil, err
		}

		if err := validateChannelConfig(channel.ChannelType, config); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		for field, value := range config {
			if IsSensitiveField(channel.ChannelType, field) && value != "" && !secretutil.IsSecretRef(value) {
				ref, err := uc.secrets.StoreSecret(ctx, "channel", channel.ID, value)
				if err != nil {
					return nil, fmt.Errorf("encrypting channel config field %s: %w", field, err)
				}

				config[field] = ref
			}
		}

		for field, oldValue := range existingConfig {
			if secretutil.IsSecretRef(oldValue) && config[field] != oldValue {
				staleRefs = append(staleRefs, oldValue)
			}
		}

		configJSON, err := json.Marshal(config)
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

	for _, ref := range staleRefs {
		_ = uc.secrets.DeleteSecret(ctx, ref) // non-fatal
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "notification_channel",
		ResourceID:    updated.ID,
		ResourceLabel: updated.Name,
		Before:        before,
		After: map[string]any{
			"name":    updated.Name,
			"enabled": updated.Enabled,
		},
	})

	return updated, nil
}

// resolveMaskedChannelSecrets returns a copy of incoming in which every masked
// sensitive field takes its stored secret ref. A sensitive field may carry a
// secret ref only when it equals the stored one.
func resolveMaskedChannelSecrets(channelType ChannelType, existing, incoming map[string]string) (map[string]string, error) {
	config := make(map[string]string, len(incoming))

	for field, value := range incoming {
		if IsSensitiveField(channelType, field) {
			switch {
			case value == secretutil.SecretMask:
				value = existing[field]
			case secretutil.IsSecretRef(value) && value != existing[field]:
				return nil, &ValidationError{Message: "unknown secret reference in " + field}
			}
		}

		config[field] = value
	}

	return config, nil
}
