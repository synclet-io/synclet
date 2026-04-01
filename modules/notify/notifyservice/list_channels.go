package notifyservice

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// ListChannelsParams holds parameters for listing notification channels.
type ListChannelsParams struct {
	WorkspaceID uuid.UUID
	ChannelType *ChannelType
}

// ListChannels returns all notification channels for a workspace.
type ListChannels struct {
	storage Storage
}

// NewListChannels creates a new ListChannels use case.
func NewListChannels(storage Storage) *ListChannels {
	return &ListChannels{storage: storage}
}

// Execute returns all notification channels for the given workspace with sensitive fields masked.
func (uc *ListChannels) Execute(ctx context.Context, params ListChannelsParams) ([]*NotificationChannel, error) {
	channelFilter := &NotificationChannelFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}

	if params.ChannelType != nil {
		channelFilter.ChannelType = filter.Equals(*params.ChannelType)
	}

	channels, err := uc.storage.NotificationChannels().Find(ctx, channelFilter)
	if err != nil {
		return nil, err
	}

	// Mask sensitive config fields in API responses.
	for _, ch := range channels {
		var config map[string]string
		if err := json.Unmarshal([]byte(ch.Config), &config); err == nil {
			masked := false

			for field := range config {
				if IsSensitiveField(ch.ChannelType, field) && config[field] != "" {
					config[field] = secretutil.SecretMask
					masked = true
				}
			}

			if masked {
				if maskedJSON, err := json.Marshal(config); err == nil {
					ch.Config = string(maskedJSON)
				}
			}
		}
	}

	return channels, nil
}
