package pipelinedestinations

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

// UpdateDestination updates an existing destination within a workspace.
type UpdateDestination struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	logger  *logging.Logger
}

// NewUpdateDestination creates a new UpdateDestination use case.
func NewUpdateDestination(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *UpdateDestination {
	return &UpdateDestination{storage: storage, secrets: secrets, logger: logger.Named("update-destination")}
}

// UpdateDestinationParams holds parameters for updating a destination.
type UpdateDestinationParams struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	Name          *string
	Config        *json.RawMessage
	RuntimeConfig *string // JSON string, nil = don't update, pointer to empty string = clear overrides
}

// Execute updates a destination.
func (uc *UpdateDestination) Execute(ctx context.Context, params UpdateDestinationParams) (*pipelineservice.Destination, error) {
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting destination: %w", err)
	}

	if params.Name != nil {
		dest.Name = *params.Name
	}
	if params.Config != nil {
		newConfig := string(*params.Config)

		// Look up connector spec via managed connector FK.
		connector, specErr := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
			ID: filter.Equals(dest.ManagedConnectorID),
		})
		if specErr != nil {
			uc.logger.WithError(specErr).WithField("managed_connector_id", dest.ManagedConnectorID).Warn(ctx, "connector spec not found, skipping secret encryption on update")
		} else {
			encryptedConfig, encErr := pipelinesecrets.UpdateConfigSecrets(ctx, uc.secrets, "destination", dest.ID, dest.Config, newConfig, connector.Spec)
			if encErr != nil {
				return nil, fmt.Errorf("encrypting destination config secrets: %w", encErr)
			}
			newConfig = encryptedConfig
		}

		dest.Config = newConfig
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
		dest.RuntimeConfig = params.RuntimeConfig
	}

	dest.UpdatedAt = time.Now()

	updated, err := uc.storage.Destinations().Update(ctx, dest)
	if err != nil {
		return nil, fmt.Errorf("updating destination: %w", err)
	}

	return updated, nil
}

// UpdateDestinationInternal updates a destination without workspace scoping.
// Used for trusted internal operations (CONTROL message config updates).
type UpdateDestinationInternal struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	logger  *logging.Logger
}

// NewUpdateDestinationInternal creates a new UpdateDestinationInternal use case.
func NewUpdateDestinationInternal(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *UpdateDestinationInternal {
	return &UpdateDestinationInternal{storage: storage, secrets: secrets, logger: logger.Named("update-destination-internal")}
}

// UpdateDestinationInternalParams holds parameters for internal destination updates.
type UpdateDestinationInternalParams struct {
	ID     uuid.UUID
	Name   *string
	Config *json.RawMessage
}

// Execute updates a destination without workspace scoping.
func (uc *UpdateDestinationInternal) Execute(ctx context.Context, params UpdateDestinationInternalParams) (*pipelineservice.Destination, error) {
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting destination: %w", err)
	}

	if params.Name != nil {
		dest.Name = *params.Name
	}
	if params.Config != nil {
		newConfig := string(*params.Config)

		// Look up connector spec via managed connector FK.
		connector, specErr := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
			ID: filter.Equals(dest.ManagedConnectorID),
		})
		if specErr != nil {
			uc.logger.WithError(specErr).WithField("managed_connector_id", dest.ManagedConnectorID).Warn(ctx, "connector spec not found, skipping secret encryption on internal update")
		} else {
			encryptedConfig, encErr := pipelinesecrets.UpdateConfigSecrets(ctx, uc.secrets, "destination", dest.ID, dest.Config, newConfig, connector.Spec)
			if encErr != nil {
				return nil, fmt.Errorf("encrypting destination config secrets: %w", encErr)
			}
			newConfig = encryptedConfig
		}

		dest.Config = newConfig
	}

	dest.UpdatedAt = time.Now()

	updated, err := uc.storage.Destinations().Update(ctx, dest)
	if err != nil {
		return nil, fmt.Errorf("updating destination: %w", err)
	}

	return updated, nil
}
