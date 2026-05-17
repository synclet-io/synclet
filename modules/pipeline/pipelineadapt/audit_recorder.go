package pipelineadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// AuditRecorder bridges the pipeline module's AuditRecorder port to the
// audit module's RecordAuditEvent use case. Actor info is pulled from the
// request context (set by the auth interceptor). When the context has no
// user (background jobs, internal callers), actor type falls back to
// "system" with a zero UUID.
type AuditRecorder struct {
	record *auditservice.RecordAuditEvent
}

func NewAuditRecorder(record *auditservice.RecordAuditEvent) *AuditRecorder {
	return &AuditRecorder{record: record}
}

var _ pipelineservice.AuditRecorder = (*AuditRecorder)(nil)

func (r *AuditRecorder) Record(ctx context.Context, event pipelineservice.AuditEvent) {
	actorType := auditservice.ActorTypeUser

	actorID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		actorType = auditservice.ActorTypeSystem
		actorID = uuid.Nil
	}

	if event.ActorID != uuid.Nil {
		actorID = event.ActorID
	}

	r.record.Execute(ctx, auditservice.RecordAuditEventParams{
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
	case "enable":
		return auditservice.ActionEnable
	case "disable":
		return auditservice.ActionDisable
	case "test":
		return auditservice.ActionTest
	case "sync":
		return auditservice.ActionSync
	default:
		return auditservice.Action(0)
	}
}

func mapResourceType(t string) auditservice.ResourceType {
	switch t {
	case "source":
		return auditservice.ResourceTypeSource
	case "destination":
		return auditservice.ResourceTypeDestination
	case "connection":
		return auditservice.ResourceTypeConnection
	case "connector":
		return auditservice.ResourceTypeConnector
	case "repository":
		return auditservice.ResourceTypeRepository
	default:
		return auditservice.ResourceType(0)
	}
}
