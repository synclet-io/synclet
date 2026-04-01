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
	logger  *logging.Logger
}

// NewUpdateSource creates a new UpdateSource use case.
func NewUpdateSource(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *UpdateSource {
	return &UpdateSource{storage: storage, secrets: secrets, logger: logger.Named("update-source")}
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
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
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
			uc.logger.WithError(specErr).WithField("managed_connector_id", src.ManagedConnectorID).Warn(ctx, "connector spec not found, skipping secret encryption on update")
		} else {
			encryptedConfig, encErr := pipelinesecrets.UpdateConfigSecrets(ctx, uc.secrets, "source", src.ID, src.Config, newConfig, connector.Spec)
			if encErr != nil {
				return nil, fmt.Errorf("encrypting source config secrets: %w", encErr)
			}

			newConfig = encryptedConfig
		}

		src.Config = newConfig
	}

	if params.RuntimeConfig != nil {
		if *params.RuntimeConfig != "" {
			parsed := pipelineservice.ParseRuntimeConfig(params.RuntimeConfig)
			if parsed != nil {
				if err := pipelineservice.ValidateRuntimeConfig(parsed); err != nil {
					return nil, fmt.Errorf("invalid runtime config: %w", err)
				}
			}
		}

		src.RuntimeConfig = params.RuntimeConfig
	}

	src.UpdatedAt = time.Now()

	updated, err := uc.storage.Sources().Update(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("updating source: %w", err)
	}

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
			uc.logger.WithError(specErr).WithField("managed_connector_id", src.ManagedConnectorID).Warn(ctx, "connector spec not found, skipping secret encryption on internal update")
		} else {
			encryptedConfig, encErr := pipelinesecrets.UpdateConfigSecrets(ctx, uc.secrets, "source", src.ID, src.Config, newConfig, connector.Spec)
			if encErr != nil {
				return nil, fmt.Errorf("encrypting source config secrets: %w", encErr)
			}

			newConfig = encryptedConfig
		}

		src.Config = newConfig
	}

	src.UpdatedAt = time.Now()

	updated, err := uc.storage.Sources().Update(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("updating source: %w", err)
	}

	return updated, nil
}
