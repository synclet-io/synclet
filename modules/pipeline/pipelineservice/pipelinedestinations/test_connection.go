package pipelinedestinations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// TestDestinationConnection tests the connectivity of a destination's connector.
type TestDestinationConnection struct {
	storage   pipelineservice.Storage
	connector pipelineservice.ConnectorClient
	secrets   pipelineservice.SecretsProvider
}

// NewTestDestinationConnection creates a new TestDestinationConnection use case.
func NewTestDestinationConnection(storage pipelineservice.Storage, connector pipelineservice.ConnectorClient, secrets pipelineservice.SecretsProvider) *TestDestinationConnection {
	return &TestDestinationConnection{
		storage:   storage,
		connector: connector,
		secrets:   secrets,
	}
}

// TestDestinationConnectionParams holds parameters for testing a destination connection.
type TestDestinationConnectionParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute tests the destination's connector configuration.
func (uc *TestDestinationConnection) Execute(ctx context.Context, params TestDestinationConnectionParams) error {
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting destination: %w", err)
	}

	// Resolve Docker image from managed connector.
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(dest.ManagedConnectorID),
	})
	if err != nil {
		return fmt.Errorf("loading managed connector: %w", err)
	}

	image := connector.DockerImage + ":" + connector.DockerTag

	// Decrypt secret references for connector operation
	decryptedConfig, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, dest.Config)
	if err != nil {
		return fmt.Errorf("decrypting config secrets: %w", err)
	}

	if err := uc.connector.Check(ctx, image, json.RawMessage(decryptedConfig)); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}
