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

// CreateDestination creates a new destination instance.
type CreateDestination struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	logger  *logging.Logger
}

// NewCreateDestination creates a new CreateDestination use case.
func NewCreateDestination(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *CreateDestination {
	return &CreateDestination{
		storage: storage,
		secrets: secrets,
		logger:  logger.Named("create-destination"),
	}
}

// CreateDestinationParams holds parameters for creating a destination.
type CreateDestinationParams struct {
	WorkspaceID        uuid.UUID
	Name               string
	ManagedConnectorID uuid.UUID
	Config             json.RawMessage
}

// Execute creates a new destination.
func (uc *CreateDestination) Execute(ctx context.Context, params CreateDestinationParams) (*pipelineservice.Destination, error) {
	// Verify managed connector exists and is ready.
	mc, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ManagedConnectorID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("managed connector not found: %w", err)
	}
	now := time.Now()
	destID := uuid.New()
	config := string(params.Config)

	// Encrypt secret fields using the managed connector's spec.
	encryptedConfig, encErr := pipelinesecrets.EncryptConfigSecrets(ctx, uc.secrets, "destination", destID, config, mc.Spec)
	if encErr != nil {
		return nil, fmt.Errorf("encrypting destination config secrets: %w", encErr)
	}
	config = encryptedConfig

	dest := &pipelineservice.Destination{
		ID:                 destID,
		WorkspaceID:        params.WorkspaceID,
		Name:               params.Name,
		ManagedConnectorID: params.ManagedConnectorID,
		Config:             config,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	created, err := uc.storage.Destinations().Create(ctx, dest)
	if err != nil {
		return nil, fmt.Errorf("creating destination: %w", err)
	}

	return created, nil
}
