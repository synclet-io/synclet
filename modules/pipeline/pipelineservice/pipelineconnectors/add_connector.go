package pipelineconnectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

// AddConnector creates a managed connector and synchronously waits for spec extraction via the task system.
type AddConnector struct {
	storage           pipelineservice.Storage
	createSpecTask    *pipelinetasks.CreateSpecTask
	waitForTaskResult *pipelinetasks.WaitForTaskResult
	audit             pipelineservice.AuditRecorder
}

// NewAddConnector creates a new AddConnector use case.
func NewAddConnector(storage pipelineservice.Storage, createSpecTask *pipelinetasks.CreateSpecTask, waitForTaskResult *pipelinetasks.WaitForTaskResult, audit pipelineservice.AuditRecorder) *AddConnector {
	return &AddConnector{storage: storage, createSpecTask: createSpecTask, waitForTaskResult: waitForTaskResult, audit: audit}
}

// AddConnectorParams holds parameters for adding a managed connector.
type AddConnectorParams struct {
	WorkspaceID   uuid.UUID
	DockerImage   string
	DockerTag     string
	Name          string
	ConnectorType pipelineservice.ConnectorType
	RepositoryID  *uuid.UUID // optional, tracks which repository the connector was added from
}

// Execute creates a managed connector, creates a spec extraction task, waits for it to complete,
// and stores the resulting spec on the connector before returning.
func (uc *AddConnector) Execute(ctx context.Context, params AddConnectorParams) (*pipelineservice.ManagedConnector, error) {
	now := time.Now()
	connector := &pipelineservice.ManagedConnector{
		ID:            uuid.New(),
		WorkspaceID:   params.WorkspaceID,
		DockerImage:   params.DockerImage,
		DockerTag:     params.DockerTag,
		Name:          params.Name,
		ConnectorType: params.ConnectorType,
		Spec:          "{}",
		CreatedAt:     now,
		UpdatedAt:     now,
		RepositoryID:  params.RepositoryID,
	}

	created, err := uc.storage.ManagedConnectors().Create(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("creating managed connector: %w", err)
	}

	withSpec, err := uc.extractSpec(ctx, params.WorkspaceID, created)
	if err != nil {
		if delErr := uc.storage.ManagedConnectors().Delete(context.WithoutCancel(ctx), &pipelineservice.ManagedConnectorFilter{
			ID:          filter.Equals(created.ID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		}); delErr != nil {
			return nil, errors.Join(err, fmt.Errorf("deleting connector without spec: %w", delErr))
		}

		return nil, err
	}

	created = withSpec

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "create",
		ResourceType:  "connector",
		ResourceID:    created.ID,
		ResourceLabel: created.Name,
		After:         connectorToAuditPayload(created),
	})

	return created, nil
}

// extractSpec runs a spec extraction task for the connector, waits for it and stores the resulting spec.
func (uc *AddConnector) extractSpec(ctx context.Context, workspaceID uuid.UUID, connector *pipelineservice.ManagedConnector) (*pipelineservice.ManagedConnector, error) {
	taskResult, err := uc.createSpecTask.Execute(ctx, pipelinetasks.CreateSpecTaskParams{
		WorkspaceID:        workspaceID,
		ManagedConnectorID: connector.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating spec task: %w", err)
	}

	result, err := uc.waitForTaskResult.Execute(ctx, pipelinetasks.WaitForTaskResultParams{
		TaskID:      taskResult.TaskID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("waiting for spec task: %w", err)
	}

	if result.Status == pipelineservice.ConnectorTaskStatusFailed {
		return nil, fmt.Errorf("spec extraction failed: %s", result.ErrorMessage)
	}

	specResult, ok := result.Result.(*pipelineservice.SpecResult)
	if !ok || specResult == nil {
		return connector, nil
	}

	specJSON, err := specResultToJSON(specResult)
	if err != nil {
		return nil, fmt.Errorf("converting spec result to JSON: %w", err)
	}

	connector.Spec = specJSON
	connector.UpdatedAt = time.Now()

	updated, err := uc.storage.ManagedConnectors().Update(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("updating connector spec: %w", err)
	}

	return updated, nil
}

func connectorToAuditPayload(connector *pipelineservice.ManagedConnector) map[string]any {
	return map[string]any{
		"name":           connector.Name,
		"docker_image":   connector.DockerImage,
		"docker_tag":     connector.DockerTag,
		"connector_type": connector.ConnectorType.String(),
	}
}

// specResultToJSON converts typed SpecResult fields back to a JSON string
// matching the protocol.ConnectorSpecification shape for ManagedConnector.Spec storage.
func specResultToJSON(specResult *pipelineservice.SpecResult) (string, error) {
	spec := map[string]any{
		"documentationUrl":      specResult.DocumentationURL,
		"changelogUrl":          specResult.ChangelogURL,
		"supportsIncremental":   specResult.SupportsIncremental,
		"supportsNormalization": specResult.SupportsNormalization,
		"supportsDBT":           specResult.SupportsDBT,
		"protocol_version":      specResult.ProtocolVersion,
	}

	// ConnectionSpecification is already a JSON string (jsonb field)
	if specResult.ConnectionSpecification != "" {
		var connSpec any
		if err := json.Unmarshal([]byte(specResult.ConnectionSpecification), &connSpec); err == nil {
			spec["connectionSpecification"] = connSpec
		}
	}

	// SupportedDestinationSyncModes is a JSON array string (jsonb field)
	if specResult.SupportedDestinationSyncModes != "" {
		var modes any
		if err := json.Unmarshal([]byte(specResult.SupportedDestinationSyncModes), &modes); err == nil {
			spec["supported_destination_sync_modes"] = modes
		}
	}

	// AdvancedAuth is a JSON object string (jsonb field)
	if specResult.AdvancedAuth != "" {
		var auth any
		if err := json.Unmarshal([]byte(specResult.AdvancedAuth), &auth); err == nil {
			spec["advancedAuth"] = auth
		}
	}

	data, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshaling spec to JSON: %w", err)
	}

	return string(data), nil
}
