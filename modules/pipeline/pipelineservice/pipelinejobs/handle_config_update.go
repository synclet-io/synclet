package pipelinejobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

// HandleConfigUpdateParams holds parameters for handling a config update from the orchestrator.
type HandleConfigUpdateParams struct {
	ConnectorType pipelineservice.ConnectorType
	ConnectorID   uuid.UUID
	Config        json.RawMessage
}

// HandleConfigUpdate processes config update reports from K8s orchestrators.
// When a connector emits a CONTROL message with updated config, the orchestrator
// forwards it here for persistence with secret encryption.
type HandleConfigUpdate struct {
	updateSourceInternal      *pipelinesources.UpdateSourceInternal
	updateDestinationInternal *pipelinedestinations.UpdateDestinationInternal
	logger                    *logging.Logger
}

// NewHandleConfigUpdate creates a new HandleConfigUpdate use case.
func NewHandleConfigUpdate(
	updateSourceInternal *pipelinesources.UpdateSourceInternal,
	updateDestinationInternal *pipelinedestinations.UpdateDestinationInternal,
	logger *logging.Logger,
) *HandleConfigUpdate {
	return &HandleConfigUpdate{
		updateSourceInternal:      updateSourceInternal,
		updateDestinationInternal: updateDestinationInternal,
		logger:                    logger.Named("handle-config-update"),
	}
}

// Execute updates the source or destination config based on connector type.
func (uc *HandleConfigUpdate) Execute(ctx context.Context, params HandleConfigUpdateParams) error {
	config := params.Config

	switch params.ConnectorType {
	case pipelineservice.ConnectorTypeSource:
		if _, err := uc.updateSourceInternal.Execute(ctx, pipelinesources.UpdateSourceInternalParams{
			ID:     params.ConnectorID,
			Config: &config,
		}); err != nil {
			return fmt.Errorf("updating source config: %w", err)
		}
	case pipelineservice.ConnectorTypeDestination:
		if _, err := uc.updateDestinationInternal.Execute(ctx, pipelinedestinations.UpdateDestinationInternalParams{
			ID:     params.ConnectorID,
			Config: &config,
		}); err != nil {
			return fmt.Errorf("updating destination config: %w", err)
		}
	default:
		return &pipelineservice.ValidationError{Message: fmt.Sprintf("invalid connector_type: %s", params.ConnectorType)}
	}

	return nil
}
