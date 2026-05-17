package workspaceadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// AuditRecorder bridges the workspace module's AuditRecorder port to the
// audit module's RecordAuditEvent use case.
type AuditRecorder struct {
	record *auditservice.RecordAuditEvent
}

func NewAuditRecorder(record *auditservice.RecordAuditEvent) *AuditRecorder {
	return &AuditRecorder{record: record}
}

var _ workspaceservice.AuditRecorder = (*AuditRecorder)(nil)

func (r *AuditRecorder) Record(ctx context.Context, event workspaceservice.AuditEvent) {
	actorType := auditservice.ActorTypeUser

	actorID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		actorType = auditservice.ActorTypeSystem
		actorID = uuid.Nil
	}

	r.record.Execute(ctx, auditservice.RecordAuditEventParams{
		WorkspaceID:   event.WorkspaceID,
		ActorType:     actorType,
		ActorID:       actorID,
		Action:        mapAction(event.Action),
		ResourceType:  mapResourceType(event.ResourceType),
		ResourceID:    event.ResourceID,
		ResourceLabel: event.ResourceLabel,
		Before:        event.Before,
		After:         event.After,
	})
}

func mapAction(a string) auditservice.Action {
	switch a {
	case "create":
		return auditservice.ActionCreate
	case "update":
		return auditservice.ActionUpdate
	case "delete":
		return auditservice.ActionDelete
	default:
		return auditservice.Action(0)
	}
}

func mapResourceType(t string) auditservice.ResourceType {
	switch t {
	case "workspace":
		return auditservice.ResourceTypeWorkspace
	case "workspace_member":
		return auditservice.ResourceTypeWorkspaceMember
	case "workspace_invite":
		return auditservice.ResourceTypeWorkspaceInvite
	default:
		return auditservice.ResourceType(0)
	}
}
