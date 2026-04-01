package notifyconnect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	notifyv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/notify/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/notify/v1/notifyv1connect"
	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

func notificationMapError(err error) error {
	var notFound notifyservice.NotFoundError
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}

	var alreadyExists notifyservice.AlreadyExistsError
	if errors.As(err, &alreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}

	var validationErr *notifyservice.ValidationError
	if errors.As(err, &validationErr) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	return err
}

var channelTypeFromProto = map[notifyv1.NotificationChannelType]notifyservice.ChannelType{
	notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_SLACK:    notifyservice.ChannelTypeSlack,
	notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL:    notifyservice.ChannelTypeEmail,
	notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM: notifyservice.ChannelTypeTelegram,
}

var channelTypeToProto = map[notifyservice.ChannelType]notifyv1.NotificationChannelType{
	notifyservice.ChannelTypeSlack:    notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_SLACK,
	notifyservice.ChannelTypeEmail:    notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL,
	notifyservice.ChannelTypeTelegram: notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_TELEGRAM,
}

var conditionFromProto = map[notifyv1.NotificationCondition]notifyservice.NotificationCondition{
	notifyv1.NotificationCondition_NOTIFICATION_CONDITION_ON_FAILURE:              notifyservice.NotificationConditionOnFailure,
	notifyv1.NotificationCondition_NOTIFICATION_CONDITION_ON_CONSECUTIVE_FAILURES: notifyservice.NotificationConditionOnConsecutiveFailures,
	notifyv1.NotificationCondition_NOTIFICATION_CONDITION_ON_ZERO_RECORDS:         notifyservice.NotificationConditionOnZeroRecords,
}

var conditionToProto = map[notifyservice.NotificationCondition]notifyv1.NotificationCondition{
	notifyservice.NotificationConditionOnFailure:             notifyv1.NotificationCondition_NOTIFICATION_CONDITION_ON_FAILURE,
	notifyservice.NotificationConditionOnConsecutiveFailures: notifyv1.NotificationCondition_NOTIFICATION_CONDITION_ON_CONSECUTIVE_FAILURES,
	notifyservice.NotificationConditionOnZeroRecords:         notifyv1.NotificationCondition_NOTIFICATION_CONDITION_ON_ZERO_RECORDS,
}

// NotificationHandler implements the NotificationService ConnectRPC handler.
type NotificationHandler struct {
	notifyv1connect.UnimplementedNotificationServiceHandler
	createChannel          *notifyservice.CreateChannel
	updateChannel          *notifyservice.UpdateChannel
	deleteChannel          *notifyservice.DeleteChannel
	listChannels           *notifyservice.ListChannels
	testChannel            *notifyservice.TestChannel
	createNotificationRule *notifyservice.CreateNotificationRule
	updateNotificationRule *notifyservice.UpdateNotificationRule
	deleteNotificationRule *notifyservice.DeleteNotificationRule
	listNotificationRules  *notifyservice.ListNotificationRules
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(
	createChannel *notifyservice.CreateChannel,
	updateChannel *notifyservice.UpdateChannel,
	deleteChannel *notifyservice.DeleteChannel,
	listChannels *notifyservice.ListChannels,
	testChannel *notifyservice.TestChannel,
	createNotificationRule *notifyservice.CreateNotificationRule,
	updateNotificationRule *notifyservice.UpdateNotificationRule,
	deleteNotificationRule *notifyservice.DeleteNotificationRule,
	listNotificationRules *notifyservice.ListNotificationRules,
) *NotificationHandler {
	return &NotificationHandler{
		createChannel:          createChannel,
		updateChannel:          updateChannel,
		deleteChannel:          deleteChannel,
		listChannels:           listChannels,
		testChannel:            testChannel,
		createNotificationRule: createNotificationRule,
		updateNotificationRule: updateNotificationRule,
		deleteNotificationRule: deleteNotificationRule,
		listNotificationRules:  listNotificationRules,
	}
}

func (h *NotificationHandler) CreateNotificationChannel(ctx context.Context, req *connect.Request[notifyv1.CreateNotificationChannelRequest]) (*connect.Response[notifyv1.CreateNotificationChannelResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if req.Msg.GetChannelType() == notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid channel_type: must be one of slack, email, telegram"))
	}

	channelType, ok := channelTypeFromProto[req.Msg.GetChannelType()]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid channel_type: must be one of slack, email, telegram"))
	}

	var config map[string]string
	if req.Msg.GetConfig() != "" {
		if err := json.Unmarshal([]byte(req.Msg.GetConfig()), &config); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config must be valid JSON: %w", err))
		}
	}

	channel, err := h.createChannel.Execute(ctx, notifyservice.CreateChannelParams{
		WorkspaceID: workspaceID,
		Name:        req.Msg.GetName(),
		ChannelType: channelType,
		Config:      config,
		Enabled:     req.Msg.GetEnabled(),
	})
	if err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.CreateNotificationChannelResponse{
		Channel: channelToProto(channel),
	}), nil
}

func (h *NotificationHandler) UpdateNotificationChannel(ctx context.Context, req *connect.Request[notifyv1.UpdateNotificationChannelRequest]) (*connect.Response[notifyv1.UpdateNotificationChannelResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := notifyservice.UpdateChannelParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}
	if req.Msg.Name != nil {
		params.Name = req.Msg.Name
	}

	if req.Msg.Config != nil {
		var config map[string]string
		if err := json.Unmarshal([]byte(req.Msg.GetConfig()), &config); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config must be valid JSON: %w", err))
		}

		params.Config = config
	}

	if req.Msg.Enabled != nil {
		params.Enabled = req.Msg.Enabled
	}

	channel, err := h.updateChannel.Execute(ctx, params)
	if err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.UpdateNotificationChannelResponse{
		Channel: channelToProto(channel),
	}), nil
}

func (h *NotificationHandler) DeleteNotificationChannel(ctx context.Context, req *connect.Request[notifyv1.DeleteNotificationChannelRequest]) (*connect.Response[notifyv1.DeleteNotificationChannelResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.deleteChannel.Execute(ctx, notifyservice.DeleteChannelParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.DeleteNotificationChannelResponse{}), nil
}

func (h *NotificationHandler) ListNotificationChannels(ctx context.Context, req *connect.Request[notifyv1.ListNotificationChannelsRequest]) (*connect.Response[notifyv1.ListNotificationChannelsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := notifyservice.ListChannelsParams{
		WorkspaceID: workspaceID,
	}

	if req.Msg.ChannelType != nil && req.Msg.GetChannelType() != notifyv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED {
		ct, ok := channelTypeFromProto[req.Msg.GetChannelType()]
		if !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid channel_type"))
		}

		params.ChannelType = &ct
	}

	channels, err := h.listChannels.Execute(ctx, params)
	if err != nil {
		return nil, notificationMapError(err)
	}

	result := make([]*notifyv1.NotificationChannel, len(channels))
	for i, ch := range channels {
		result[i] = channelToProto(ch)
	}

	return connect.NewResponse(&notifyv1.ListNotificationChannelsResponse{
		Channels: result,
		Total:    int32(len(channels)),
	}), nil
}

func (h *NotificationHandler) TestNotificationChannel(ctx context.Context, req *connect.Request[notifyv1.TestNotificationChannelRequest]) (*connect.Response[notifyv1.TestNotificationChannelResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.testChannel.Execute(ctx, notifyservice.TestChannelParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.TestNotificationChannelResponse{}), nil
}

func (h *NotificationHandler) CreateNotificationRule(ctx context.Context, req *connect.Request[notifyv1.CreateNotificationRuleRequest]) (*connect.Response[notifyv1.CreateNotificationRuleResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	channelID, err := uuid.Parse(req.Msg.GetChannelId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid channel_id: %w", err))
	}

	var connectionID uuid.UUID
	if req.Msg.ConnectionId != nil {
		connectionID, err = uuid.Parse(req.Msg.GetConnectionId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
		}
	}

	if req.Msg.GetCondition() == notifyv1.NotificationCondition_NOTIFICATION_CONDITION_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid condition: must be one of on_failure, on_consecutive_failures, on_zero_records"))
	}

	cond, ok := conditionFromProto[req.Msg.GetCondition()]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid condition: must be one of on_failure, on_consecutive_failures, on_zero_records"))
	}

	rule, err := h.createNotificationRule.Execute(ctx, notifyservice.CreateNotificationRuleParams{
		WorkspaceID:    workspaceID,
		ChannelID:      channelID,
		ConnectionID:   connectionID,
		Condition:      cond,
		ConditionValue: int(req.Msg.GetConditionValue()),
		Enabled:        req.Msg.GetEnabled(),
	})
	if err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.CreateNotificationRuleResponse{
		Rule: ruleToProto(rule),
	}), nil
}

func (h *NotificationHandler) UpdateNotificationRule(ctx context.Context, req *connect.Request[notifyv1.UpdateNotificationRuleRequest]) (*connect.Response[notifyv1.UpdateNotificationRuleResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := notifyservice.UpdateNotificationRuleParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}

	if req.Msg.Condition != nil && req.Msg.GetCondition() != notifyv1.NotificationCondition_NOTIFICATION_CONDITION_UNSPECIFIED {
		cond, ok := conditionFromProto[req.Msg.GetCondition()]
		if !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid condition"))
		}

		params.Condition = &cond
	}

	if req.Msg.ConditionValue != nil {
		val := int(req.Msg.GetConditionValue())
		params.ConditionValue = &val
	}

	if req.Msg.Enabled != nil {
		params.Enabled = req.Msg.Enabled
	}

	rule, err := h.updateNotificationRule.Execute(ctx, params)
	if err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.UpdateNotificationRuleResponse{
		Rule: ruleToProto(rule),
	}), nil
}

func (h *NotificationHandler) DeleteNotificationRule(ctx context.Context, req *connect.Request[notifyv1.DeleteNotificationRuleRequest]) (*connect.Response[notifyv1.DeleteNotificationRuleResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.deleteNotificationRule.Execute(ctx, notifyservice.DeleteNotificationRuleParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, notificationMapError(err)
	}

	return connect.NewResponse(&notifyv1.DeleteNotificationRuleResponse{}), nil
}

func (h *NotificationHandler) ListNotificationRules(ctx context.Context, req *connect.Request[notifyv1.ListNotificationRulesRequest]) (*connect.Response[notifyv1.ListNotificationRulesResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := notifyservice.ListNotificationRulesParams{
		WorkspaceID: workspaceID,
	}

	if req.Msg.ChannelId != nil {
		channelID, err := uuid.Parse(req.Msg.GetChannelId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid channel_id: %w", err))
		}

		params.ChannelID = &channelID
	}

	if req.Msg.ConnectionId != nil {
		connectionID, err := uuid.Parse(req.Msg.GetConnectionId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
		}

		params.ConnectionID = &connectionID
	}

	rules, err := h.listNotificationRules.Execute(ctx, params)
	if err != nil {
		return nil, notificationMapError(err)
	}

	result := make([]*notifyv1.NotificationRule, len(rules))
	for i, rule := range rules {
		result[i] = ruleToProto(rule)
	}

	return connect.NewResponse(&notifyv1.ListNotificationRulesResponse{
		Rules: result,
		Total: int32(len(rules)),
	}), nil
}

func channelToProto(ch *notifyservice.NotificationChannel) *notifyv1.NotificationChannel {
	return &notifyv1.NotificationChannel{
		Id:          ch.ID.String(),
		WorkspaceId: ch.WorkspaceID.String(),
		Name:        ch.Name,
		ChannelType: channelTypeToProto[ch.ChannelType],
		Config:      ch.Config,
		Enabled:     ch.Enabled,
		CreatedAt:   timestamppb.New(ch.CreatedAt),
		UpdatedAt:   timestamppb.New(ch.UpdatedAt),
	}
}

func ruleToProto(rule *notifyservice.NotificationRule) *notifyv1.NotificationRule {
	proto := &notifyv1.NotificationRule{
		Id:             rule.ID.String(),
		WorkspaceId:    rule.WorkspaceID.String(),
		ChannelId:      rule.ChannelID.String(),
		Condition:      conditionToProto[rule.Condition],
		ConditionValue: int32(rule.ConditionValue),
		Enabled:        rule.Enabled,
		CreatedAt:      timestamppb.New(rule.CreatedAt),
		UpdatedAt:      timestamppb.New(rule.UpdatedAt),
	}

	if rule.ConnectionID != uuid.Nil {
		connID := rule.ConnectionID.String()
		proto.ConnectionId = &connID
	}

	return proto
}
