package pipelinetasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// ClaimTaskResult bundles everything an executor needs to run a connector task.
type ClaimTaskResult struct {
	TaskID      uuid.UUID
	TaskType    pipelineservice.ConnectorTaskType
	Image       string
	Config      []byte // Decrypted JSON config (nil for spec tasks)
	WorkspaceID uuid.UUID
	// ServiceAccountName is the service_account_name of the source's or
	// destination's runtime config; empty for spec and inline-config checks.
	ServiceAccountName string
	// RuntimeConfig is the raw runtime_config of the source or destination;
	// empty for spec and inline-config checks.
	RuntimeConfig string
}

// ClaimTask atomically claims a pending connector task and resolves image + decrypted config.
type ClaimTask struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
}

// NewClaimTask creates a new ClaimTask use case.
func NewClaimTask(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider) *ClaimTask {
	return &ClaimTask{
		storage: storage,
		secrets: secrets,
	}
}

// Execute claims the next pending task, resolves image and config, and returns the bundle.
// Returns nil, nil when no pending tasks are available.
func (uc *ClaimTask) Execute(ctx context.Context, workerID string) (*ClaimTaskResult, error) {
	// 1. Atomically claim the next pending task.
	task, err := uc.storage.ConnectorTasks().ClaimPendingTask(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("claiming pending task: %w", err)
	}

	if task == nil {
		return nil, nil
	}

	// 2. Resolve image, config and runtime settings based on payload type.
	var target claimTarget

	switch payload := task.Payload.(type) {
	case *pipelineservice.CheckPayload:
		target, err = uc.resolveCheckPayload(ctx, task.WorkspaceID, payload)
	case *pipelineservice.SpecPayload:
		target.image, err = uc.resolveSpecPayload(ctx, task.WorkspaceID, payload)
	case *pipelineservice.DiscoverPayload:
		target, err = uc.resolveDiscoverPayload(ctx, task.WorkspaceID, payload)
	default:
		return nil, fmt.Errorf("unknown payload type: %T", task.Payload)
	}

	if err != nil {
		return nil, err
	}

	return &ClaimTaskResult{
		TaskID:      task.ID,
		TaskType:    task.TaskType,
		Image:       target.image,
		Config:      target.config,
		WorkspaceID: task.WorkspaceID,

		ServiceAccountName: target.serviceAccount,
		RuntimeConfig:      target.runtimeConfig,
	}, nil
}

// claimTarget is what a claimed task runs: the connector image, its decrypted
// config and the runtime settings of the task's source or destination.
type claimTarget struct {
	image          string
	config         []byte
	serviceAccount string
	runtimeConfig  string
}

// entityRuntimeConfig returns an entity's raw runtime config, empty when unset.
func entityRuntimeConfig(runtimeConfig *string) string {
	if runtimeConfig == nil {
		return ""
	}

	return *runtimeConfig
}

// entityServiceAccount returns the service_account_name of an entity's runtime config.
func entityServiceAccount(runtimeConfig *string) string {
	cfg := pipelineservice.ParseRuntimeConfig(runtimeConfig)
	if cfg == nil {
		return ""
	}

	return cfg.ServiceAccountName
}

// resolveCheckPayload resolves the target of a check task.
// CheckPayload may contain inline config or reference a source/destination.
func (uc *ClaimTask) resolveCheckPayload(ctx context.Context, workspaceID uuid.UUID, payload *pipelineservice.CheckPayload) (claimTarget, error) {
	var target claimTarget

	// Resolve managed connector for image, scoped to workspace.
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(payload.ManagedConnectorID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return claimTarget{}, fmt.Errorf("loading managed connector: %w", err)
	}

	target.image = connector.DockerImage + ":" + connector.DockerTag

	// Resolve config: use inline config if present, otherwise load from source/destination.
	var configJSON string
	if payload.Config != nil {
		configJSON = *payload.Config
		target.serviceAccount = payload.ServiceAccountName
	} else if payload.SourceID != nil {
		src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
			ID:          filter.Equals(*payload.SourceID),
			WorkspaceID: filter.Equals(workspaceID),
		})
		if err != nil {
			return claimTarget{}, fmt.Errorf("loading source: %w", err)
		}

		configJSON = src.Config
		target.serviceAccount = entityServiceAccount(src.RuntimeConfig)
		target.runtimeConfig = entityRuntimeConfig(src.RuntimeConfig)
	} else if payload.DestinationID != nil {
		dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
			ID:          filter.Equals(*payload.DestinationID),
			WorkspaceID: filter.Equals(workspaceID),
		})
		if err != nil {
			return claimTarget{}, fmt.Errorf("loading destination: %w", err)
		}

		configJSON = dest.Config
		target.serviceAccount = entityServiceAccount(dest.RuntimeConfig)
		target.runtimeConfig = entityRuntimeConfig(dest.RuntimeConfig)
	}

	// Decrypt config secrets.
	decrypted, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, configJSON)
	if err != nil {
		return claimTarget{}, fmt.Errorf("decrypting config secrets: %w", err)
	}

	target.config = []byte(decrypted)

	return target, nil
}

// resolveSpecPayload resolves image for a spec task (no config needed).
func (uc *ClaimTask) resolveSpecPayload(ctx context.Context, workspaceID uuid.UUID, payload *pipelineservice.SpecPayload) (string, error) {
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(payload.ManagedConnectorID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return "", fmt.Errorf("loading managed connector: %w", err)
	}

	return connector.DockerImage + ":" + connector.DockerTag, nil
}

// resolveDiscoverPayload resolves the target of a discover task.
func (uc *ClaimTask) resolveDiscoverPayload(ctx context.Context, workspaceID uuid.UUID, payload *pipelineservice.DiscoverPayload) (claimTarget, error) {
	var target claimTarget

	// Resolve managed connector for image, scoped to workspace.
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(payload.ManagedConnectorID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return claimTarget{}, fmt.Errorf("loading managed connector: %w", err)
	}

	target.image = connector.DockerImage + ":" + connector.DockerTag

	// Resolve source config, scoped to workspace.
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(payload.SourceID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return claimTarget{}, fmt.Errorf("loading source: %w", err)
	}

	// Decrypt config secrets.
	decrypted, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, src.Config)
	if err != nil {
		return claimTarget{}, fmt.Errorf("decrypting config secrets: %w", err)
	}

	target.config = []byte(decrypted)
	target.serviceAccount = entityServiceAccount(src.RuntimeConfig)
	target.runtimeConfig = entityRuntimeConfig(src.RuntimeConfig)

	return target, nil
}
