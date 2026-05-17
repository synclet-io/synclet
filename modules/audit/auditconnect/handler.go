package auditconnect

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/audit/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/audit/v1/auditv1connect"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// Handler implements the AuditService ConnectRPC handler.
type Handler struct {
	auditv1connect.UnimplementedAuditServiceHandler
	listAuditEvents *auditservice.ListAuditEvents
	getAuditEvent   *auditservice.GetAuditEvent
}

func NewHandler(listAuditEvents *auditservice.ListAuditEvents, getAuditEvent *auditservice.GetAuditEvent) *Handler {
	return &Handler{
		listAuditEvents: listAuditEvents,
		getAuditEvent:   getAuditEvent,
	}
}

func (h *Handler) ListAuditEvents(ctx context.Context, req *connect.Request[auditv1.ListAuditEventsRequest]) (*connect.Response[auditv1.ListAuditEventsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := auditservice.ListAuditEventsParams{
		WorkspaceID: workspaceID,
		Limit:       int(req.Msg.GetLimit()),
		Offset:      int(req.Msg.GetOffset()),
	}

	if actorID := req.Msg.GetActorId(); actorID != "" {
		id, parseErr := uuid.Parse(actorID)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}

		params.ActorID = &id
	}

	if resourceID := req.Msg.GetResourceId(); resourceID != "" {
		id, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}

		params.ResourceID = &id
	}

	if rt := req.Msg.GetResourceType(); rt != auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_UNSPECIFIED {
		mapped := protoResourceTypeToDomain(rt)
		params.ResourceType = &mapped
	}

	if action := req.Msg.GetAction(); action != auditv1.AuditAction_AUDIT_ACTION_UNSPECIFIED {
		mapped := protoActionToDomain(action)
		params.Action = &mapped
	}

	if since := req.Msg.GetSince(); since != nil {
		t := since.AsTime()
		params.Since = &t
	}

	if until := req.Msg.GetUntil(); until != nil {
		t := until.AsTime()
		params.Until = &t
	}

	events, err := h.listAuditEvents.Execute(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &auditv1.ListAuditEventsResponse{
		Events: make([]*auditv1.AuditEvent, 0, len(events)),
	}
	for _, e := range events {
		resp.Events = append(resp.Events, eventToProto(e))
	}

	return connect.NewResponse(resp), nil
}

func (h *Handler) GetAuditEvent(ctx context.Context, req *connect.Request[auditv1.GetAuditEventRequest]) (*connect.Response[auditv1.GetAuditEventResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	event, err := h.getAuditEvent.Execute(ctx, auditservice.GetAuditEventParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&auditv1.GetAuditEventResponse{Event: eventToProto(event)}), nil
}

func eventToProto(event *auditservice.AuditEvent) *auditv1.AuditEvent {
	var createdAt *timestamppb.Timestamp
	if !event.CreatedAt.IsZero() && !event.CreatedAt.Before(time.Unix(0, 0)) {
		createdAt = timestamppb.New(event.CreatedAt)
	}

	return &auditv1.AuditEvent{
		Id:            event.ID.String(),
		WorkspaceId:   event.WorkspaceID.String(),
		ActorType:     domainActorTypeToProto(event.ActorType),
		ActorId:       event.ActorID.String(),
		ActorLabel:    event.ActorLabel,
		Action:        domainActionToProto(event.Action),
		ResourceType:  domainResourceTypeToProto(event.ResourceType),
		ResourceId:    event.ResourceID.String(),
		ResourceLabel: event.ResourceLabel,
		DiffJson:      event.DiffJSON,
		DiffTruncated: event.DiffTruncated,
		CreatedAt:     createdAt,
	}
}

func domainActorTypeToProto(t auditservice.ActorType) auditv1.AuditActorType {
	switch t {
	case auditservice.ActorTypeUser:
		return auditv1.AuditActorType_AUDIT_ACTOR_TYPE_USER
	case auditservice.ActorTypeAPIKey:
		return auditv1.AuditActorType_AUDIT_ACTOR_TYPE_API_KEY
	case auditservice.ActorTypeSystem:
		return auditv1.AuditActorType_AUDIT_ACTOR_TYPE_SYSTEM
	default:
		return auditv1.AuditActorType_AUDIT_ACTOR_TYPE_UNSPECIFIED
	}
}

func domainActionToProto(a auditservice.Action) auditv1.AuditAction {
	switch a {
	case auditservice.ActionCreate:
		return auditv1.AuditAction_AUDIT_ACTION_CREATE
	case auditservice.ActionUpdate:
		return auditv1.AuditAction_AUDIT_ACTION_UPDATE
	case auditservice.ActionDelete:
		return auditv1.AuditAction_AUDIT_ACTION_DELETE
	case auditservice.ActionEnable:
		return auditv1.AuditAction_AUDIT_ACTION_ENABLE
	case auditservice.ActionDisable:
		return auditv1.AuditAction_AUDIT_ACTION_DISABLE
	case auditservice.ActionTest:
		return auditv1.AuditAction_AUDIT_ACTION_TEST
	case auditservice.ActionSync:
		return auditv1.AuditAction_AUDIT_ACTION_SYNC
	case auditservice.ActionLogin:
		return auditv1.AuditAction_AUDIT_ACTION_LOGIN
	case auditservice.ActionLogout:
		return auditv1.AuditAction_AUDIT_ACTION_LOGOUT
	default:
		return auditv1.AuditAction_AUDIT_ACTION_UNSPECIFIED
	}
}

func protoActionToDomain(a auditv1.AuditAction) auditservice.Action {
	switch a {
	case auditv1.AuditAction_AUDIT_ACTION_CREATE:
		return auditservice.ActionCreate
	case auditv1.AuditAction_AUDIT_ACTION_UPDATE:
		return auditservice.ActionUpdate
	case auditv1.AuditAction_AUDIT_ACTION_DELETE:
		return auditservice.ActionDelete
	case auditv1.AuditAction_AUDIT_ACTION_ENABLE:
		return auditservice.ActionEnable
	case auditv1.AuditAction_AUDIT_ACTION_DISABLE:
		return auditservice.ActionDisable
	case auditv1.AuditAction_AUDIT_ACTION_TEST:
		return auditservice.ActionTest
	case auditv1.AuditAction_AUDIT_ACTION_SYNC:
		return auditservice.ActionSync
	case auditv1.AuditAction_AUDIT_ACTION_LOGIN:
		return auditservice.ActionLogin
	case auditv1.AuditAction_AUDIT_ACTION_LOGOUT:
		return auditservice.ActionLogout
	default:
		return auditservice.Action(0)
	}
}

func domainResourceTypeToProto(rt auditservice.ResourceType) auditv1.AuditResourceType {
	switch rt {
	case auditservice.ResourceTypeSource:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_SOURCE
	case auditservice.ResourceTypeDestination:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_DESTINATION
	case auditservice.ResourceTypeConnection:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_CONNECTION
	case auditservice.ResourceTypeConnector:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_CONNECTOR
	case auditservice.ResourceTypeRepository:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_REPOSITORY
	case auditservice.ResourceTypeNotificationChannel:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_NOTIFICATION_CHANNEL
	case auditservice.ResourceTypeNotificationRule:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_NOTIFICATION_RULE
	case auditservice.ResourceTypeWebhook:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WEBHOOK
	case auditservice.ResourceTypeWorkspace:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WORKSPACE
	case auditservice.ResourceTypeWorkspaceMember:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WORKSPACE_MEMBER
	case auditservice.ResourceTypeWorkspaceInvite:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WORKSPACE_INVITE
	case auditservice.ResourceTypeAPIKey:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_API_KEY
	case auditservice.ResourceTypeUser:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_USER
	default:
		return auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_UNSPECIFIED
	}
}

func protoResourceTypeToDomain(rt auditv1.AuditResourceType) auditservice.ResourceType {
	switch rt {
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_SOURCE:
		return auditservice.ResourceTypeSource
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_DESTINATION:
		return auditservice.ResourceTypeDestination
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_CONNECTION:
		return auditservice.ResourceTypeConnection
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_CONNECTOR:
		return auditservice.ResourceTypeConnector
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_REPOSITORY:
		return auditservice.ResourceTypeRepository
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_NOTIFICATION_CHANNEL:
		return auditservice.ResourceTypeNotificationChannel
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_NOTIFICATION_RULE:
		return auditservice.ResourceTypeNotificationRule
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WEBHOOK:
		return auditservice.ResourceTypeWebhook
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WORKSPACE:
		return auditservice.ResourceTypeWorkspace
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WORKSPACE_MEMBER:
		return auditservice.ResourceTypeWorkspaceMember
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_WORKSPACE_INVITE:
		return auditservice.ResourceTypeWorkspaceInvite
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_API_KEY:
		return auditservice.ResourceTypeAPIKey
	case auditv1.AuditResourceType_AUDIT_RESOURCE_TYPE_USER:
		return auditservice.ResourceTypeUser
	default:
		return auditservice.ResourceType(0)
	}
}
