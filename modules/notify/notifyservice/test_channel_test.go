package notifyservice

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingAuditRecorder struct {
	events []AuditEvent
}

func (r *capturingAuditRecorder) Record(_ context.Context, event AuditEvent) {
	r.events = append(r.events, event)
}

func TestTestChannel_RecordsAuditEvent(t *testing.T) {
	wsID := uuid.New()
	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: wsID, Name: "ops-slack", ChannelType: ChannelTypeSlack}
	store := newTestStorage([]*NotificationChannel{channel}, nil)
	deliverer := &mockChannelDeliverer{}
	recorder := &capturingAuditRecorder{}

	uc := NewTestChannel(store, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: deliverer}, recorder)
	result, err := uc.Execute(context.Background(), TestChannelParams{ID: channel.ID, WorkspaceID: wsID})
	require.NoError(t, err)
	assert.Equal(t, TestChannelResult{Delivered: true}, result)

	require.Len(t, recorder.events, 1)
	event := recorder.events[0]
	assert.Equal(t, wsID, event.WorkspaceID)
	assert.Equal(t, "test", event.Action)
	assert.Equal(t, "notification_channel", event.ResourceType)
	assert.Equal(t, channel.ID, event.ResourceID)
	assert.Equal(t, "ops-slack", event.ResourceLabel)
	assert.Equal(t, map[string]any{"delivered": true}, event.After)
}

func TestTestChannel_RecordsFailedDelivery(t *testing.T) {
	wsID := uuid.New()
	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: wsID, Name: "ops-slack", ChannelType: ChannelTypeSlack}
	store := newTestStorage([]*NotificationChannel{channel}, nil)
	deliverer := &mockChannelDeliverer{err: errors.New("boom")}
	recorder := &capturingAuditRecorder{}

	uc := NewTestChannel(store, map[ChannelType]ChannelDeliverer{ChannelTypeSlack: deliverer}, recorder)
	result, err := uc.Execute(context.Background(), TestChannelParams{ID: channel.ID, WorkspaceID: wsID})
	require.NoError(t, err)
	assert.Equal(t, TestChannelResult{DeliveryError: "boom"}, result)

	require.Len(t, recorder.events, 1)
	assert.Equal(t, map[string]any{"delivered": false, "delivery_error": "boom"}, recorder.events[0].After)
}

func TestTestChannel_HidesRequestURL(t *testing.T) {
	wsID := uuid.New()
	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: wsID, Name: "ops-telegram", ChannelType: ChannelTypeTelegram}
	store := newTestStorage([]*NotificationChannel{channel}, nil)
	deliverer := &mockChannelDeliverer{err: fmt.Errorf("sending telegram notification: %w", &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/bot123:SECRET/sendMessage",
		Err: errors.New("dial tcp: i/o timeout"),
	})}
	recorder := &capturingAuditRecorder{}

	uc := NewTestChannel(store, map[ChannelType]ChannelDeliverer{ChannelTypeTelegram: deliverer}, recorder)
	result, err := uc.Execute(context.Background(), TestChannelParams{ID: channel.ID, WorkspaceID: wsID})
	require.NoError(t, err)
	assert.False(t, result.Delivered)
	assert.NotContains(t, result.DeliveryError, "SECRET")
	assert.Contains(t, result.DeliveryError, "i/o timeout")
	assert.NotContains(t, recorder.events[0].After.(map[string]any)["delivery_error"], "SECRET")
}
