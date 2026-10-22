package pipelinesources

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// UpdateSource updates an existing source within a workspace.
type UpdateSource struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	audit   pipelineservice.AuditRecorder
	logger  *logging.Logger
}

// NewUpdateSource creates a new UpdateSource use case.
func NewUpdateSource(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, audit pipelineservice.AuditRecorder, logger *logging.Logger) *UpdateSource {
	return &UpdateSource{storage: storage, secrets: secrets, audit: audit, logger: logger.Named("update-source")}
}

// UpdateSourceParams holds parameters for updating a source.
type UpdateSourceParams struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	Name          *string
	Config        *json.RawMessage
	RuntimeConfig *string // JSON string, nil = don't update, pointer to empty string = clear overrides
}

// Execute updates a source.
func (uc *UpdateSource) Execute(ctx context.Context, params UpdateSourceParams) (*pipelineservice.Source, error) {
	if params.RuntimeConfig != nil && *params.RuntimeConfig != "" {
		if _, err := pipelineservice.DecodeRuntimeConfig(*params.RuntimeConfig); err != nil {
			return nil, err
		}
	}

	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting source: %w", err)
	}

	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(src.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connector spec: %w", err)
	}

	before := map[string]any{
		"name":           src.Name,
		"config":         auditConfig(src.Config, connector.Spec),
		"runtime_config": auditRuntimeConfig(src.RuntimeConfig),
	}

	if params.Name != nil {
		src.Name = *params.Name
	}

	if params.Config != nil {
		newConfig := string(*params.Config)

		encryptedConfig, encErr := pipelinesecrets.UpdateConfigSecrets(ctx, uc.secrets, "source", src.ID, src.Config, newConfig, connector.Spec)
		if encErr != nil {
			return nil, fmt.Errorf("encrypting source config secrets: %w", encErr)
		}

		newConfig = encryptedConfig

		src.Config = newConfig
	}

	if params.RuntimeConfig != nil {
		src.RuntimeConfig = params.RuntimeConfig
	}

	src.UpdatedAt = time.Now()

	updated, err := uc.storage.Sources().Update(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("updating source: %w", err)
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "source",
		ResourceID:    updated.ID,
		ResourceLabel: updated.Name,
		Before:        before,
		After: map[string]any{
			"name":           updated.Name,
			"config":         auditConfig(updated.Config, connector.Spec),
			"runtime_config": auditRuntimeConfig(updated.RuntimeConfig),
		},
	})

	return updated, nil
}

// UpdateSourceInternal updates a source without workspace scoping.
// Used for trusted internal operations (CONTROL message config updates).
type UpdateSourceInternal struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	logger  *logging.Logger
}

// NewUpdateSourceInternal creates a new UpdateSourceInternal use case.
func NewUpdateSourceInternal(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *UpdateSourceInternal {
	return &UpdateSourceInternal{storage: storage, secrets: secrets, logger: logger.Named("update-source-internal")}
}

// UpdateSourceInternalParams holds parameters for internal source updates.
type UpdateSourceInternalParams struct {
	ID     uuid.UUID
	Name   *string
	Config *json.RawMessage
}

// Execute updates a source without workspace scoping.
func (uc *UpdateSourceInternal) Execute(ctx context.Context, params UpdateSourceInternalParams) (*pipelineservice.Source, error) {
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting source: %w", err)
	}

	if params.Name != nil {
		src.Name = *params.Name
	}

	if params.Config != nil {
		newConfig := string(*params.Config)

		// Look up connector spec via managed connector FK.
		connector, specErr := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
			ID: filter.Equals(src.ManagedConnectorID),
		})
		if specErr != nil {
			return nil, fmt.Errorf("loading connector spec for secret encryption: %w", specErr)
		}

		encryptedConfig, encErr := pipelinesecrets.UpdateConfigSecrets(ctx, uc.secrets, "source", src.ID, src.Config, newConfig, connector.Spec)
		if encErr != nil {
			return nil, fmt.Errorf("encrypting source config secrets: %w", encErr)
		}

		newConfig = encryptedConfig

		src.Config = newConfig
	}

	src.UpdatedAt = time.Now()

	updated, err := uc.storage.Sources().Update(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("updating source: %w", err)
	}

	return updated, nil
}
