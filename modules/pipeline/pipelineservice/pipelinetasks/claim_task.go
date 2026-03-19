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

	// 2. Resolve image and config based on payload type.
	var image string
	var config []byte

	switch p := task.Payload.(type) {
	case *pipelineservice.CheckPayload:
		image, config, err = uc.resolveCheckPayload(ctx, task.WorkspaceID, p)
	case *pipelineservice.SpecPayload:
		image, err = uc.resolveSpecPayload(ctx, task.WorkspaceID, p)
	case *pipelineservice.DiscoverPayload:
		image, config, err = uc.resolveDiscoverPayload(ctx, task.WorkspaceID, p)
	default:
		return nil, fmt.Errorf("unknown payload type: %T", task.Payload)
	}
	if err != nil {
		return nil, err
	}

	return &ClaimTaskResult{
		TaskID:      task.ID,
		TaskType:    task.TaskType,
		Image:       image,
		Config:      config,
		WorkspaceID: task.WorkspaceID,
	}, nil
}

// resolveCheckPayload resolves image and config for a check task.
// CheckPayload may contain inline config or reference a source/destination.
func (uc *ClaimTask) resolveCheckPayload(ctx context.Context, workspaceID uuid.UUID, p *pipelineservice.CheckPayload) (image string, config []byte, err error) {
	// Resolve managed connector for image, scoped to workspace.
	mc, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(p.ManagedConnectorID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return "", nil, fmt.Errorf("loading managed connector: %w", err)
	}
	image = mc.DockerImage + ":" + mc.DockerTag

	// Resolve config: use inline config if present, otherwise load from source/destination.
	var configJSON string
	if p.Config != nil {
		configJSON = *p.Config
	} else if p.SourceID != nil {
		src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
			ID:          filter.Equals(*p.SourceID),
			WorkspaceID: filter.Equals(workspaceID),
		})
		if err != nil {
			return "", nil, fmt.Errorf("loading source: %w", err)
		}
		configJSON = src.Config
	} else if p.DestinationID != nil {
		dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
			ID:          filter.Equals(*p.DestinationID),
			WorkspaceID: filter.Equals(workspaceID),
		})
		if err != nil {
			return "", nil, fmt.Errorf("loading destination: %w", err)
		}
		configJSON = dest.Config
	}

	// Decrypt config secrets.
	decrypted, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, configJSON)
	if err != nil {
		return "", nil, fmt.Errorf("decrypting config secrets: %w", err)
	}

	return image, []byte(decrypted), nil
}

// resolveSpecPayload resolves image for a spec task (no config needed).
func (uc *ClaimTask) resolveSpecPayload(ctx context.Context, workspaceID uuid.UUID, p *pipelineservice.SpecPayload) (string, error) {
	mc, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(p.ManagedConnectorID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return "", fmt.Errorf("loading managed connector: %w", err)
	}
	return mc.DockerImage + ":" + mc.DockerTag, nil
}

// resolveDiscoverPayload resolves image and config for a discover task.
func (uc *ClaimTask) resolveDiscoverPayload(ctx context.Context, workspaceID uuid.UUID, p *pipelineservice.DiscoverPayload) (image string, config []byte, err error) {
	// Resolve managed connector for image, scoped to workspace.
	mc, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(p.ManagedConnectorID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return "", nil, fmt.Errorf("loading managed connector: %w", err)
	}
	image = mc.DockerImage + ":" + mc.DockerTag

	// Resolve source config, scoped to workspace.
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(p.SourceID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return "", nil, fmt.Errorf("loading source: %w", err)
	}

	// Decrypt config secrets.
	decrypted, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, src.Config)
	if err != nil {
		return "", nil, fmt.Errorf("decrypting config secrets: %w", err)
	}

	return image, []byte(decrypted), nil
}
