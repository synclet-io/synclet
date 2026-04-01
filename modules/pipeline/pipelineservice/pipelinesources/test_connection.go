package pipelinesources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// TestSourceConnection tests the connectivity of a source's connector.
type TestSourceConnection struct {
	storage   pipelineservice.Storage
	connector pipelineservice.ConnectorClient
	secrets   pipelineservice.SecretsProvider
}

// NewTestSourceConnection creates a new TestSourceConnection use case.
func NewTestSourceConnection(storage pipelineservice.Storage, connector pipelineservice.ConnectorClient, secrets pipelineservice.SecretsProvider) *TestSourceConnection {
	return &TestSourceConnection{
		storage:   storage,
		connector: connector,
		secrets:   secrets,
	}
}

// TestSourceConnectionParams holds parameters for testing a source connection.
type TestSourceConnectionParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute tests the source's connector configuration.
func (uc *TestSourceConnection) Execute(ctx context.Context, params TestSourceConnectionParams) error {
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting source: %w", err)
	}

	// Resolve Docker image from managed connector.
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(src.ManagedConnectorID),
	})
	if err != nil {
		return fmt.Errorf("loading managed connector: %w", err)
	}

	image := connector.DockerImage + ":" + connector.DockerTag

	// Decrypt secret references for connector operation
	decryptedConfig, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, src.Config)
	if err != nil {
		return fmt.Errorf("decrypting config secrets: %w", err)
	}

	if err := uc.connector.Check(ctx, image, json.RawMessage(decryptedConfig)); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}
