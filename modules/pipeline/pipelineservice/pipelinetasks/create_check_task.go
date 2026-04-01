package pipelinetasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// CreateCheckTask creates a connector check task for a source or destination.
type CreateCheckTask struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
}

// NewCreateCheckTask creates a new CreateCheckTask use case.
func NewCreateCheckTask(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider) *CreateCheckTask {
	return &CreateCheckTask{storage: storage, secrets: secrets}
}

// CreateCheckTaskParams holds parameters for creating a check task.
type CreateCheckTaskParams struct {
	WorkspaceID uuid.UUID

	// Option 1: Test existing source/destination by ID.
	SourceID      *uuid.UUID
	DestinationID *uuid.UUID

	// Option 2: Test with raw config (no source/destination created yet).
	ManagedConnectorID *uuid.UUID
	Config             json.RawMessage // Plaintext config from the frontend.
}

// Execute creates a connector check task and returns the task ID.
func (uc *CreateCheckTask) Execute(ctx context.Context, params CreateCheckTaskParams) (*CreateTaskResult, error) {
	hasEntityRef := params.SourceID != nil || params.DestinationID != nil
	hasDirectConfig := params.ManagedConnectorID != nil && len(params.Config) > 0

	if !hasEntityRef && !hasDirectConfig {
		return nil, errors.New("either source_id/destination_id or managed_connector_id+config must be provided")
	}

	taskID := uuid.New()
	payload := &pipelineservice.CheckPayload{}

	if hasDirectConfig {
		// Direct config path: encrypt secrets and store config in the payload.
		connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
			ID:          filter.Equals(*params.ManagedConnectorID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		})
		if err != nil {
			return nil, fmt.Errorf("managed connector not found: %w", err)
		}

		encrypted, err := pipelinesecrets.EncryptConfigSecrets(ctx, uc.secrets, "connector_task", taskID, string(params.Config), connector.Spec)
		if err != nil {
			return nil, fmt.Errorf("encrypting config secrets: %w", err)
		}

		payload.ManagedConnectorID = connector.ID
		payload.Config = &encrypted
	} else {
		// Entity reference path: resolve managed connector ID from source/destination.
		if params.SourceID != nil {
			src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
				ID:          filter.Equals(*params.SourceID),
				WorkspaceID: filter.Equals(params.WorkspaceID),
			})
			if err != nil {
				return nil, fmt.Errorf("finding source: %w", err)
			}

			payload.SourceID = params.SourceID
			payload.ManagedConnectorID = src.ManagedConnectorID
		} else {
			dst, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
				ID:          filter.Equals(*params.DestinationID),
				WorkspaceID: filter.Equals(params.WorkspaceID),
			})
			if err != nil {
				return nil, fmt.Errorf("finding destination: %w", err)
			}

			payload.DestinationID = params.DestinationID
			payload.ManagedConnectorID = dst.ManagedConnectorID
		}
	}

	task := &pipelineservice.ConnectorTask{
		ID:          taskID,
		WorkspaceID: params.WorkspaceID,
		TaskType:    pipelineservice.ConnectorTaskTypeCheck,
		Status:      pipelineservice.ConnectorTaskStatusPending,
		Payload:     payload,
	}

	if _, err := uc.storage.ConnectorTasks().Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating check task: %w", err)
	}

	return &CreateTaskResult{TaskID: task.ID}, nil
}
