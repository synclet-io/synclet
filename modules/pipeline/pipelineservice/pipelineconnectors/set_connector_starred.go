package pipelineconnectors

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// SetConnectorStarred sets or clears the workspace-wide star of a managed connector.
type SetConnectorStarred struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewSetConnectorStarred creates a new SetConnectorStarred use case.
func NewSetConnectorStarred(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *SetConnectorStarred {
	return &SetConnectorStarred{storage: storage, audit: audit}
}

// SetConnectorStarredParams holds parameters for starring or unstarring a managed connector.
type SetConnectorStarredParams struct {
	ConnectorID uuid.UUID
	WorkspaceID uuid.UUID
	Starred     bool
}

// Execute stores the star flag of the workspace's managed connector.
func (uc *SetConnectorStarred) Execute(ctx context.Context, params SetConnectorStarredParams) (*pipelineservice.ManagedConnector, error) {
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ConnectorID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading managed connector: %w", err)
	}

	before := starredAuditPayload(connector)

	if err := uc.storage.ManagedConnectors().SetStarred(ctx, params.WorkspaceID, params.ConnectorID, params.Starred); err != nil {
		return nil, fmt.Errorf("setting managed connector star: %w", err)
	}

	updated := connector.Copy()
	updated.Starred = params.Starred
	updated.UpdatedAt = time.Now()

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "connector",
		ResourceID:    updated.ID,
		ResourceLabel: updated.Name,
		Before:        before,
		After:         starredAuditPayload(&updated),
	})

	return &updated, nil
}

func starredAuditPayload(connector *pipelineservice.ManagedConnector) map[string]any {
	payload := connectorToAuditPayload(connector)
	payload["starred"] = connector.Starred

	return payload
}
