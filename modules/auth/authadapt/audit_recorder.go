package authadapt

import (
	"context"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type auditEventWriter interface {
	Execute(ctx context.Context, params auditservice.RecordAuditEventParams)
}

type userWorkspacesLister interface {
	Execute(ctx context.Context, userID uuid.UUID) ([]workspaceservice.UserWorkspace, error)
}

// AuditRecorder bridges the auth module's AuditRecorder port to the audit
// module's RecordAuditEvent use case. Events without a workspace are
// recorded once in each workspace the actor is a member of.
type AuditRecorder struct {
	record     auditEventWriter
	workspaces userWorkspacesLister
	logger     *logging.Logger
}

func NewAuditRecorder(record *auditservice.RecordAuditEvent, workspaces *workspaceservice.ListWorkspacesForUser, logger *logging.Logger) *AuditRecorder {
	return &AuditRecorder{record: record, workspaces: workspaces, logger: logger.Named("auth-audit-recorder")}
}

var _ authservice.AuditRecorder = (*AuditRecorder)(nil)

func (r *AuditRecorder) Record(ctx context.Context, event authservice.AuditEvent) {
	actorType := auditservice.ActorTypeUser
	actorID := event.ActorID

	if actorID == uuid.Nil {
		ctxUserID, err := connectutil.UserIDFromContext(ctx)
		if err != nil {
			actorType = auditservice.ActorTypeSystem
		} else {
			actorID = ctxUserID
		}
	}

	params := auditservice.RecordAuditEventParams{
		WorkspaceID:   event.WorkspaceID,
		ActorType:     actorType,
		ActorID:       actorID,
		ActorLabel:    event.ActorLabel,
		Action:        mapAction(event.Action),
		ResourceType:  mapResourceType(event.ResourceType),
		ResourceID:    event.ResourceID,
		ResourceLabel: event.ResourceLabel,
		Before:        event.Before,
		After:         event.After,
	}

	if event.WorkspaceID != uuid.Nil {
		r.record.Execute(ctx, params)

		return
	}

	if actorID == uuid.Nil {
		return
	}

	userWorkspaces, err := r.workspaces.Execute(ctx, actorID)
	if err != nil {
		r.logger.WithError(err).WithField("user_id", actorID).Error(ctx, "listing user workspaces for audit event failed")

		return
	}

	for _, uw := range userWorkspaces {
		params.WorkspaceID = uw.Workspace.ID
		r.record.Execute(ctx, params)
	}
}

func mapAction(a string) auditservice.Action {
	switch a {
	case "create":
		return auditservice.ActionCreate
	case "update":
		return auditservice.ActionUpdate
	case "delete":
		return auditservice.ActionDelete
	case "login":
		return auditservice.ActionLogin
	case "logout":
		return auditservice.ActionLogout
	default:
		return auditservice.Action(0)
	}
}

func mapResourceType(t string) auditservice.ResourceType {
	switch t {
	case "user":
		return auditservice.ResourceTypeUser
	case "api_key":
		return auditservice.ResourceTypeAPIKey
	default:
		return auditservice.ResourceType(0)
	}
}
