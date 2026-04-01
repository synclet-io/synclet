package notifyconnect

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	webhookv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/webhook/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/webhook/v1/webhookv1connect"
	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

func mapError(err error) error {
	var notFound notifyservice.NotFoundError
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}

	var alreadyExists notifyservice.AlreadyExistsError
	if errors.As(err, &alreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}

	var validation *notifyservice.ValidationError
	if errors.As(err, &validation) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	return err
}

// Handler implements the WebhookService ConnectRPC handler.
type Handler struct {
	webhookv1connect.UnimplementedWebhookServiceHandler
	createWebhook *notifyservice.CreateWebhook
	updateWebhook *notifyservice.UpdateWebhook
	deleteWebhook *notifyservice.DeleteWebhook
	listWebhooks  *notifyservice.ListWebhooks
}

// NewHandler creates a new webhook handler.
func NewHandler(
	createWebhook *notifyservice.CreateWebhook,
	updateWebhook *notifyservice.UpdateWebhook,
	deleteWebhook *notifyservice.DeleteWebhook,
	listWebhooks *notifyservice.ListWebhooks,
) *Handler {
	return &Handler{
		createWebhook: createWebhook,
		updateWebhook: updateWebhook,
		deleteWebhook: deleteWebhook,
		listWebhooks:  listWebhooks,
	}
}

func (h *Handler) CreateWebhook(ctx context.Context, req *connect.Request[webhookv1.CreateWebhookRequest]) (*connect.Response[webhookv1.CreateWebhookResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "url", Value: req.Msg.GetUrl(), MaxLen: connectutil.MaxURLLength},
		connectutil.StringValidation{Field: "secret", Value: req.Msg.GetSecret(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := connectutil.ValidateWebhookURL(req.Msg.GetUrl()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	webhook, err := h.createWebhook.Execute(ctx, notifyservice.CreateWebhookParams{
		WorkspaceID: workspaceID,
		URL:         req.Msg.GetUrl(),
		Events:      req.Msg.GetEvents(),
		Secret:      req.Msg.GetSecret(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&webhookv1.CreateWebhookResponse{
		Webhook: webhookToProto(webhook),
	}), nil
}

func (h *Handler) UpdateWebhook(ctx context.Context, req *connect.Request[webhookv1.UpdateWebhookRequest]) (*connect.Response[webhookv1.UpdateWebhookResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := notifyservice.UpdateWebhookParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Events:      req.Msg.GetEvents(),
	}
	if req.Msg.Url != nil {
		if err := connectutil.ValidateWebhookURL(req.Msg.GetUrl()); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		params.URL = req.Msg.Url
	}

	if req.Msg.Enabled != nil {
		params.Enabled = req.Msg.Enabled
	}

	webhook, err := h.updateWebhook.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&webhookv1.UpdateWebhookResponse{
		Webhook: webhookToProto(webhook),
	}), nil
}

func (h *Handler) DeleteWebhook(ctx context.Context, req *connect.Request[webhookv1.DeleteWebhookRequest]) (*connect.Response[webhookv1.DeleteWebhookResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.deleteWebhook.Execute(ctx, notifyservice.DeleteWebhookParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&webhookv1.DeleteWebhookResponse{}), nil
}

func (h *Handler) ListWebhooks(ctx context.Context, req *connect.Request[webhookv1.ListWebhooksRequest]) (*connect.Response[webhookv1.ListWebhooksResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	webhooks, err := h.listWebhooks.Execute(ctx, notifyservice.ListWebhooksParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]*webhookv1.WebhookInfo, len(webhooks))
	for i, wh := range webhooks {
		result[i] = webhookToProto(wh)
	}

	return connect.NewResponse(&webhookv1.ListWebhooksResponse{
		Webhooks: result,
	}), nil
}

func webhookToProto(webhook *notifyservice.Webhook) *webhookv1.WebhookInfo {
	var events []string
	if err := json.Unmarshal([]byte(webhook.Events), &events); err != nil {
		events = []string{} // Explicit empty on corruption.
	}

	return &webhookv1.WebhookInfo{
		Id:          webhook.ID.String(),
		WorkspaceId: webhook.WorkspaceID.String(),
		Url:         webhook.URL,
		Events:      events,
		Enabled:     webhook.Enabled,
		CreatedAt:   timestamppb.New(webhook.CreatedAt),
		UpdatedAt:   timestamppb.New(webhook.UpdatedAt),
	}
}
