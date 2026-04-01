package workspaceconnect

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	workspacev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1/workspacev1connect"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// Handler implements the WorkspaceService ConnectRPC handler.
type Handler struct {
	workspacev1connect.UnimplementedWorkspaceServiceHandler
	createWorkspace       *workspaceservice.CreateWorkspace
	updateWorkspace       *workspaceservice.UpdateWorkspace
	deleteWorkspace       *workspaceservice.DeleteWorkspace
	getWorkspace          *workspaceservice.GetWorkspace
	listWorkspacesForUser *workspaceservice.ListWorkspacesForUser
	removeMember          *workspaceservice.RemoveMember
	listMembers           *workspaceservice.ListMembers

	// Invite use cases
	createInvite     *workspaceservice.CreateInvite
	acceptInvite     *workspaceservice.AcceptInvite
	declineInvite    *workspaceservice.DeclineInvite
	revokeInvite     *workspaceservice.RevokeInvite
	resendInvite     *workspaceservice.ResendInvite
	listInvites      *workspaceservice.ListInvites
	getInviteByToken *workspaceservice.GetInviteByToken
}

// NewHandler creates a new workspace handler.
func NewHandler(
	createWorkspace *workspaceservice.CreateWorkspace,
	updateWorkspace *workspaceservice.UpdateWorkspace,
	deleteWorkspace *workspaceservice.DeleteWorkspace,
	getWorkspace *workspaceservice.GetWorkspace,
	listWorkspacesForUser *workspaceservice.ListWorkspacesForUser,
	removeMember *workspaceservice.RemoveMember,
	listMembers *workspaceservice.ListMembers,
	createInvite *workspaceservice.CreateInvite,
	acceptInvite *workspaceservice.AcceptInvite,
	declineInvite *workspaceservice.DeclineInvite,
	revokeInvite *workspaceservice.RevokeInvite,
	resendInvite *workspaceservice.ResendInvite,
	listInvites *workspaceservice.ListInvites,
	getInviteByToken *workspaceservice.GetInviteByToken,
) *Handler {
	return &Handler{
		createWorkspace:       createWorkspace,
		updateWorkspace:       updateWorkspace,
		deleteWorkspace:       deleteWorkspace,
		getWorkspace:          getWorkspace,
		listWorkspacesForUser: listWorkspacesForUser,
		removeMember:          removeMember,
		listMembers:           listMembers,
		createInvite:          createInvite,
		acceptInvite:          acceptInvite,
		declineInvite:         declineInvite,
		revokeInvite:          revokeInvite,
		resendInvite:          resendInvite,
		listInvites:           listInvites,
		getInviteByToken:      getInviteByToken,
	}
}

func (h *Handler) CreateWorkspace(ctx context.Context, req *connect.Request[workspacev1.CreateWorkspaceRequest]) (*connect.Response[workspacev1.CreateWorkspaceResponse], error) {
	ownerID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	workspace, err := h.createWorkspace.Execute(ctx, req.Msg.GetName(), ownerID)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.CreateWorkspaceResponse{
		Workspace: workspaceToProto(workspace),
	}), nil
}

func (h *Handler) UpdateWorkspace(ctx context.Context, req *connect.Request[workspacev1.UpdateWorkspaceRequest]) (*connect.Response[workspacev1.UpdateWorkspaceResponse], error) {
	// Use workspace ID from auth context to prevent IDOR -- the role interceptor
	// validates admin role against this same context workspace, so we must target it.
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := workspaceservice.UpdateWorkspaceParams{
		ID: workspaceID,
	}

	if req.Msg.GetName() != "" {
		if err := connectutil.ValidateStringLengths(
			connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
		); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		params.Name = &req.Msg.Name
	}

	workspace, err := h.updateWorkspace.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.UpdateWorkspaceResponse{
		Workspace: workspaceToProto(workspace),
	}), nil
}

func (h *Handler) DeleteWorkspace(ctx context.Context, req *connect.Request[workspacev1.DeleteWorkspaceRequest]) (*connect.Response[workspacev1.DeleteWorkspaceResponse], error) {
	// Use workspace ID from auth context to prevent IDOR -- the role interceptor
	// validates admin role against this same context workspace, so we must target it.
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := h.deleteWorkspace.Execute(ctx, workspaceID); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.DeleteWorkspaceResponse{}), nil
}

func (h *Handler) GetWorkspace(ctx context.Context, req *connect.Request[workspacev1.GetWorkspaceRequest]) (*connect.Response[workspacev1.GetWorkspaceResponse], error) {
	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	workspace, err := h.getWorkspace.Execute(ctx, id, &userID)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.GetWorkspaceResponse{
		Workspace: workspaceToProto(workspace),
	}), nil
}

func (h *Handler) ListWorkspaces(ctx context.Context, req *connect.Request[workspacev1.ListWorkspacesRequest]) (*connect.Response[workspacev1.ListWorkspacesResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	workspaces, err := h.listWorkspacesForUser.Execute(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]*workspacev1.WorkspaceInfo, len(workspaces))
	for i, ws := range workspaces {
		result[i] = workspaceToProto(ws)
	}

	return connect.NewResponse(&workspacev1.ListWorkspacesResponse{
		Workspaces: result,
	}), nil
}

func (h *Handler) RemoveMember(ctx context.Context, req *connect.Request[workspacev1.RemoveMemberRequest]) (*connect.Response[workspacev1.RemoveMemberResponse], error) {
	// Use workspace ID from auth context to prevent IDOR -- the role interceptor
	// validates admin role against this same context workspace, so we must target it.
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	userID, err := uuid.Parse(req.Msg.GetUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.removeMember.Execute(ctx, workspaceID, userID); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.RemoveMemberResponse{}), nil
}

func (h *Handler) ListMembers(ctx context.Context, req *connect.Request[workspacev1.ListMembersRequest]) (*connect.Response[workspacev1.ListMembersResponse], error) {
	// Use workspace ID from auth context to prevent IDOR.
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	members, err := h.listMembers.Execute(ctx, workspaceID, &userID)
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]*workspacev1.WorkspaceMemberInfo, len(members))
	for i, m := range members {
		result[i] = memberToProto(m)
	}

	return connect.NewResponse(&workspacev1.ListMembersResponse{
		Members: result,
	}), nil
}

// CreateInvite creates a workspace invite and sends an email.
func (h *Handler) CreateInvite(ctx context.Context, req *connect.Request[workspacev1.CreateInviteRequest]) (*connect.Response[workspacev1.CreateInviteResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace ID required"))
	}

	role := protoToMemberRole(req.Msg.GetRole())
	if !role.IsValid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid role: %v", req.Msg.GetRole()))
	}

	invite, err := h.createInvite.Execute(ctx, workspaceservice.CreateInviteParams{
		WorkspaceID:   workspaceID,
		InviterUserID: userID,
		Email:         req.Msg.GetEmail(),
		Role:          role,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.CreateInviteResponse{
		Invite: inviteToProto(invite, "", ""),
	}), nil
}

// AcceptInvite accepts an invite by token. Requires authentication.
func (h *Handler) AcceptInvite(ctx context.Context, req *connect.Request[workspacev1.AcceptInviteRequest]) (*connect.Response[workspacev1.AcceptInviteResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	// Resolve user email for D-14 enforcement.
	userEmail, err := connectutil.EmailFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("cannot determine user email"))
	}

	result, err := h.acceptInvite.Execute(ctx, req.Msg.GetToken(), userID, userEmail)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.AcceptInviteResponse{
		WorkspaceId:   result.WorkspaceID.String(),
		WorkspaceName: result.WorkspaceName,
	}), nil
}

// DeclineInvite declines an invite by token. No auth required.
func (h *Handler) DeclineInvite(ctx context.Context, req *connect.Request[workspacev1.DeclineInviteRequest]) (*connect.Response[workspacev1.DeclineInviteResponse], error) {
	if err := h.declineInvite.Execute(ctx, req.Msg.GetToken()); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.DeclineInviteResponse{}), nil
}

// RevokeInvite revokes a pending invite. Requires admin auth.
func (h *Handler) RevokeInvite(ctx context.Context, req *connect.Request[workspacev1.RevokeInviteRequest]) (*connect.Response[workspacev1.RevokeInviteResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace ID required"))
	}

	inviteID, err := uuid.Parse(req.Msg.GetInviteId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.revokeInvite.Execute(ctx, inviteID, workspaceID); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.RevokeInviteResponse{}), nil
}

// ResendInvite resends an invite email and resets TTL. Requires admin auth.
func (h *Handler) ResendInvite(ctx context.Context, req *connect.Request[workspacev1.ResendInviteRequest]) (*connect.Response[workspacev1.ResendInviteResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace ID required"))
	}

	inviteID, err := uuid.Parse(req.Msg.GetInviteId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.resendInvite.Execute(ctx, inviteID, workspaceID); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&workspacev1.ResendInviteResponse{}), nil
}

// ListInvites lists all invites for the current workspace.
func (h *Handler) ListInvites(ctx context.Context, req *connect.Request[workspacev1.ListInvitesRequest]) (*connect.Response[workspacev1.ListInvitesResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace ID required"))
	}

	invites, err := h.listInvites.Execute(ctx, workspaceID)
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]*workspacev1.WorkspaceInviteInfo, len(invites))
	for i, inv := range invites {
		result[i] = inviteToProto(inv, "", "")
	}

	return connect.NewResponse(&workspacev1.ListInvitesResponse{
		Invites: result,
	}), nil
}

// GetInviteByToken retrieves invite info by token. No auth required.
func (h *Handler) GetInviteByToken(ctx context.Context, req *connect.Request[workspacev1.GetInviteByTokenRequest]) (*connect.Response[workspacev1.GetInviteByTokenResponse], error) {
	result, err := h.getInviteByToken.Execute(ctx, req.Msg.GetToken())
	if err != nil {
		return nil, mapError(err)
	}

	info := inviteByTokenResultToProto(result)

	return connect.NewResponse(&workspacev1.GetInviteByTokenResponse{
		Invite: info,
	}), nil
}

// inviteByTokenResultToProto maps the UC result to proto, handling expired status.
func inviteByTokenResultToProto(result *workspaceservice.InviteByTokenResult) *workspacev1.WorkspaceInviteInfo {
	info := inviteToProto(result.Invite, result.WorkspaceName, result.InviterName)
	if result.IsExpired {
		info.Status = workspacev1.InviteStatus_INVITE_STATUS_EXPIRED
	}

	return info
}

// Proto conversion helpers

func workspaceToProto(workspace *workspaceservice.Workspace) *workspacev1.WorkspaceInfo {
	return &workspacev1.WorkspaceInfo{
		Id:        workspace.ID.String(),
		Name:      workspace.Name,
		Slug:      workspace.Slug,
		CreatedAt: timestamppb.New(workspace.CreatedAt),
		UpdatedAt: timestamppb.New(workspace.UpdatedAt),
	}
}

func memberToProto(member *workspaceservice.WorkspaceMember) *workspacev1.WorkspaceMemberInfo {
	return &workspacev1.WorkspaceMemberInfo{
		Id:          member.ID.String(),
		WorkspaceId: member.WorkspaceID.String(),
		UserId:      member.UserID.String(),
		Role:        memberRoleToProto(member.Role),
		JoinedAt:    timestamppb.New(member.JoinedAt),
	}
}

func inviteToProto(inv *workspaceservice.WorkspaceInvite, workspaceName, inviterName string) *workspacev1.WorkspaceInviteInfo {
	return &workspacev1.WorkspaceInviteInfo{
		Id:            inv.ID.String(),
		WorkspaceId:   inv.WorkspaceID.String(),
		WorkspaceName: workspaceName,
		InviterUserId: inv.InviterUserID.String(),
		InviterName:   inviterName,
		Email:         inv.Email,
		Role:          memberRoleToProto(inv.Role),
		Status:        inviteStatusToProto(inv.Status),
		ExpiresAt:     timestamppb.New(inv.ExpiresAt),
		CreatedAt:     timestamppb.New(inv.CreatedAt),
	}
}

func protoToMemberRole(r workspacev1.MemberRole) workspaceservice.MemberRole {
	switch r {
	case workspacev1.MemberRole_MEMBER_ROLE_ADMIN:
		return workspaceservice.MemberRoleAdmin
	case workspacev1.MemberRole_MEMBER_ROLE_EDITOR:
		return workspaceservice.MemberRoleEditor
	case workspacev1.MemberRole_MEMBER_ROLE_VIEWER:
		return workspaceservice.MemberRoleViewer
	default:
		return 0
	}
}

func memberRoleToProto(r workspaceservice.MemberRole) workspacev1.MemberRole {
	switch r {
	case workspaceservice.MemberRoleAdmin:
		return workspacev1.MemberRole_MEMBER_ROLE_ADMIN
	case workspaceservice.MemberRoleEditor:
		return workspacev1.MemberRole_MEMBER_ROLE_EDITOR
	case workspaceservice.MemberRoleViewer:
		return workspacev1.MemberRole_MEMBER_ROLE_VIEWER
	default:
		return workspacev1.MemberRole_MEMBER_ROLE_UNSPECIFIED
	}
}

func inviteStatusToProto(s workspaceservice.InviteStatus) workspacev1.InviteStatus {
	switch s {
	case workspaceservice.InviteStatusPending:
		return workspacev1.InviteStatus_INVITE_STATUS_PENDING
	case workspaceservice.InviteStatusAccepted:
		return workspacev1.InviteStatus_INVITE_STATUS_ACCEPTED
	case workspaceservice.InviteStatusDeclined:
		return workspacev1.InviteStatus_INVITE_STATUS_DECLINED
	case workspaceservice.InviteStatusRevoked:
		return workspacev1.InviteStatus_INVITE_STATUS_REVOKED
	default:
		return workspacev1.InviteStatus_INVITE_STATUS_UNSPECIFIED
	}
}

// mapError maps domain errors to ConnectRPC error codes.
func mapError(err error) error {
	var notFound workspaceservice.NotFoundError
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}

	var alreadyExists workspaceservice.AlreadyExistsError
	if errors.As(err, &alreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}

	var validationErr *workspaceservice.ValidationError
	if errors.As(err, &validationErr) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	return err
}
