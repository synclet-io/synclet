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

// CreateSource creates a new source instance.
type CreateSource struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	logger  *logging.Logger
}

// NewCreateSource creates a new CreateSource use case.
func NewCreateSource(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, logger *logging.Logger) *CreateSource {
	return &CreateSource{
		storage: storage,
		secrets: secrets,
		logger:  logger.Named("create-source"),
	}
}

// CreateSourceParams holds parameters for creating a source.
type CreateSourceParams struct {
	WorkspaceID        uuid.UUID
	Name               string
	ManagedConnectorID uuid.UUID
	Config             json.RawMessage
}

// Execute creates a new source.
func (uc *CreateSource) Execute(ctx context.Context, params CreateSourceParams) (*pipelineservice.Source, error) {
	// Verify managed connector exists and is ready.
	mc, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ManagedConnectorID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("managed connector not found: %w", err)
	}
	now := time.Now()
	srcID := uuid.New()
	config := string(params.Config)

	// Encrypt secret fields using the managed connector's spec.
	encryptedConfig, encErr := pipelinesecrets.EncryptConfigSecrets(ctx, uc.secrets, "source", srcID, config, mc.Spec)
	if encErr != nil {
		return nil, fmt.Errorf("encrypting source config secrets: %w", encErr)
	}
	config = encryptedConfig

	src := &pipelineservice.Source{
		ID:                 srcID,
		WorkspaceID:        params.WorkspaceID,
		Name:               params.Name,
		ManagedConnectorID: params.ManagedConnectorID,
		Config:             config,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	created, err := uc.storage.Sources().Create(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("creating source: %w", err)
	}

	return created, nil
}
