package notifyeventhandle

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// recordingTransport answers every request with 200 and keeps its URL and body.
type recordingTransport struct {
	mu       sync.Mutex
	requests map[string][]byte
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	rt.mu.Lock()
	rt.requests[req.URL.String()] = body
	rt.mu.Unlock()

	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(http.NoBody), Header: http.Header{}}, nil
}

type recordingChannel struct {
	deliveries map[uuid.UUID]notifyservice.WebhookEvent
}

func (c *recordingChannel) Deliver(_ context.Context, channel *notifyservice.NotificationChannel, event notifyservice.WebhookEvent) error {
	c.deliveries[channel.ID] = event

	return nil
}

// notifyStore keeps webhooks, channels and rules in memory and applies the
// rule filter's workspace, enabled and connection conditions.
type notifyStore struct {
	notifyservice.Storage

	webhooks []*notifyservice.Webhook
	channels []*notifyservice.NotificationChannel
	rules    []*notifyservice.NotificationRule
}

func (s *notifyStore) Webhooks() notifyservice.WebhooksStorage { return webhookTable{notifyStore: s} }

func (s *notifyStore) NotificationChannels() notifyservice.NotificationChannelsStorage {
	return channelTable{notifyStore: s}
}

func (s *notifyStore) NotificationRules() notifyservice.NotificationRulesStorage {
	return ruleTable{notifyStore: s}
}

type webhookTable struct {
	notifyservice.WebhooksStorage

	*notifyStore
}

func (t webhookTable) Find(_ context.Context, f *notifyservice.WebhookFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*notifyservice.Webhook, error) {
	var result []*notifyservice.Webhook

	for _, webhook := range t.webhooks {
		if matches(f.WorkspaceID, webhook.WorkspaceID) && matches(f.Enabled, webhook.Enabled) {
			result = append(result, webhook)
		}
	}

	return result, nil
}

type channelTable struct {
	notifyservice.NotificationChannelsStorage

	*notifyStore
}

func (t channelTable) First(_ context.Context, f *notifyservice.NotificationChannelFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*notifyservice.NotificationChannel, error) {
	for _, channel := range t.channels {
		if matches(f.ID, channel.ID) && matches(f.WorkspaceID, channel.WorkspaceID) {
			return channel, nil
		}
	}

	return nil, notifyservice.ErrNotificationChannelNotFound
}

type ruleTable struct {
	notifyservice.NotificationRulesStorage

	*notifyStore
}

func (t ruleTable) Find(_ context.Context, f *notifyservice.NotificationRuleFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*notifyservice.NotificationRule, error) {
	var result []*notifyservice.NotificationRule

	for _, rule := range t.rules {
		if !matches(f.WorkspaceID, rule.WorkspaceID) || !matches(f.Enabled, rule.Enabled) || !matches(f.ConnectionID, rule.ConnectionID) {
			continue
		}

		if len(f.Or) > 0 && !slices.ContainsFunc(f.Or, func(or *notifyservice.NotificationRuleFilter) bool {
			return matches(or.ConnectionID, rule.ConnectionID)
		}) {
			continue
		}

		result = append(result, rule)
	}

	return result, nil
}

func matches[T comparable](f filter.Filter[T], value T) bool {
	switch typed := f.(type) {
	case nil:
		return true
	case *filter.EqualsFilter[T]:
		return typed.Value == value
	case *filter.InFilter[T]:
		return slices.Contains(typed.Values, value)
	default:
		panic("unsupported filter in test store")
	}
}

type workflowFailureFixture struct {
	handler   *WorkflowRunEventHandler
	transport *recordingTransport
	channels  *recordingChannel

	workspaceID uuid.UUID
	stepConnA   uuid.UUID
	stepConnB   uuid.UUID

	subscribedChannel   uuid.UUID
	workspaceChannel    uuid.UUID
	otherChannel        uuid.UUID
	subscribedWebhook   string
	wildcardWebhook     string
	unsubscribedWebhook string
}

func newWorkflowFailureFixture() *workflowFailureFixture {
	fx := &workflowFailureFixture{
		transport:           &recordingTransport{requests: map[string][]byte{}},
		channels:            &recordingChannel{deliveries: map[uuid.UUID]notifyservice.WebhookEvent{}},
		workspaceID:         uuid.New(),
		stepConnA:           uuid.New(),
		stepConnB:           uuid.New(),
		subscribedChannel:   uuid.New(),
		workspaceChannel:    uuid.New(),
		otherChannel:        uuid.New(),
		subscribedWebhook:   "https://8.8.8.8/workflow",
		wildcardWebhook:     "https://8.8.8.8/all",
		unsubscribedWebhook: "https://8.8.8.8/sync",
	}

	webhook := func(url, events string) *notifyservice.Webhook {
		return &notifyservice.Webhook{ID: uuid.New(), WorkspaceID: fx.workspaceID, URL: url, Events: events, Enabled: true}
	}
	channel := func(id uuid.UUID) *notifyservice.NotificationChannel {
		return &notifyservice.NotificationChannel{ID: id, WorkspaceID: fx.workspaceID, ChannelType: notifyservice.ChannelTypeSlack, Enabled: true}
	}
	rule := func(channelID, connectionID uuid.UUID) *notifyservice.NotificationRule {
		return &notifyservice.NotificationRule{
			ID: uuid.New(), WorkspaceID: fx.workspaceID, ChannelID: channelID, ConnectionID: connectionID,
			Condition: notifyservice.NotificationConditionOnFailure, Enabled: true,
		}
	}

	store := &notifyStore{
		webhooks: []*notifyservice.Webhook{
			webhook(fx.subscribedWebhook, `["workflow.failed"]`),
			webhook(fx.wildcardWebhook, `["*"]`),
			webhook(fx.unsubscribedWebhook, `["sync.failed"]`),
		},
		channels: []*notifyservice.NotificationChannel{
			channel(fx.subscribedChannel), channel(fx.workspaceChannel), channel(fx.otherChannel),
		},
		rules: []*notifyservice.NotificationRule{
			rule(fx.subscribedChannel, fx.stepConnB),
			rule(fx.workspaceChannel, uuid.Nil),
			rule(fx.otherChannel, uuid.New()),
		},
	}

	logger := &logging.Logger{Delegate: noopLogDelegate{}}
	cfg := notifyservice.Config{WebhookMaxRetries: 1}

	fx.handler = NewWorkflowRunEventHandler(
		logger,
		notifyservice.NewDeliverWebhookWithClient(store, nil, cfg, &http.Client{Transport: fx.transport}),
		notifyservice.NewDeliverNotification(store, map[notifyservice.ChannelType]notifyservice.ChannelDeliverer{
			notifyservice.ChannelTypeSlack: fx.channels,
		}, logger),
	)

	return fx
}

func (fx *workflowFailureFixture) finished(status pipelineservice.WorkflowRunStatus, reason *string) *pipelineservice.WorkflowRunFinishedEventData {
	return &pipelineservice.WorkflowRunFinishedEventData{
		WorkspaceID:   fx.workspaceID,
		WorkflowID:    uuid.New(),
		WorkflowName:  "nightly",
		RunID:         uuid.New(),
		Status:        status,
		Error:         reason,
		ConnectionIDs: []uuid.UUID{fx.stepConnA, fx.stepConnB},
	}
}

func finishedMessage(t *testing.T, data *pipelineservice.WorkflowRunFinishedEventData) *pipelineservice.PipelineEvent {
	t.Helper()

	return &pipelineservice.PipelineEvent{EventType: pipelineservice.PipelineEventTypeWorkflowRunFinished, Data: data}
}

func TestWorkflowRunEventHandler_FailedRunNotifiesSubscribers(t *testing.T) {
	fx := newWorkflowFailureFixture()
	reason := "boom"
	data := fx.finished(pipelineservice.WorkflowRunStatusFailed, &reason)

	require.NoError(t, fx.handler.Handle(encode(t, finishedMessage(t, data))))

	require.Contains(t, fx.transport.requests, fx.subscribedWebhook, "a webhook subscribed to workflow.failed gets it")
	require.Contains(t, fx.transport.requests, fx.wildcardWebhook, "a webhook subscribed to * gets it")
	assert.NotContains(t, fx.transport.requests, fx.unsubscribedWebhook, "a webhook not subscribed to workflow.failed does not")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(fx.transport.requests[fx.subscribedWebhook], &payload))
	assert.Equal(t, "workflow.failed", payload["event"])
	assert.Equal(t, fx.workspaceID.String(), payload["workspace_id"])
	assert.Equal(t, data.WorkflowID.String(), payload["workflow_id"])
	assert.Equal(t, "nightly", payload["workflow_name"])
	assert.Equal(t, data.RunID.String(), payload["workflow_run_id"])
	assert.Equal(t, "boom", payload["error"])

	require.Contains(t, fx.channels.deliveries, fx.subscribedChannel, "a channel whose rule covers a step's connection is notified")
	require.Contains(t, fx.channels.deliveries, fx.workspaceChannel, "a workspace-wide failure rule covers workflows")
	assert.NotContains(t, fx.channels.deliveries, fx.otherChannel, "a rule on a connection outside the workflow does not fire")
	assert.Len(t, fx.channels.deliveries, 2)

	notification := fx.channels.deliveries[fx.subscribedChannel]
	assert.Equal(t, "workflow.failed", notification.Event)
	assert.Equal(t, data.WorkflowID.String(), notification.WorkflowID)
	assert.Equal(t, "nightly", notification.WorkflowName)
	assert.Equal(t, data.RunID.String(), notification.WorkflowRunID)
	assert.Equal(t, "boom", notification.Error)
}

func TestWorkflowRunEventHandler_SucceededRunSendsNothing(t *testing.T) {
	fx := newWorkflowFailureFixture()

	require.NoError(t, fx.handler.Handle(encode(t, finishedMessage(t, fx.finished(pipelineservice.WorkflowRunStatusSucceeded, nil)))))

	assert.Empty(t, fx.transport.requests)
	assert.Empty(t, fx.channels.deliveries)
}
