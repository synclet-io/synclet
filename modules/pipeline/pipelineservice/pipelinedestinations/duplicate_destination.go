package pipelinedestinations

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// DuplicateDestination copies a destination into another executor group.
type DuplicateDestination struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	audit   pipelineservice.AuditRecorder
}

// NewDuplicateDestination creates a new DuplicateDestination use case.
func NewDuplicateDestination(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, audit pipelineservice.AuditRecorder) *DuplicateDestination {
	return &DuplicateDestination{storage: storage, secrets: secrets, audit: audit}
}

// DuplicateDestinationParams holds parameters for duplicating a destination.
type DuplicateDestinationParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	// ExecutorGroup is the group of the copy; empty means default.
	ExecutorGroup string
	// Name is the name of the copy; empty keeps the original's name.
	Name string
}

// Execute creates a destination with the original's managed connector, config and
// runtime config in params.ExecutorGroup. The copy's secrets are stored anew
// under its own id; connections and their state stay with the original.
func (uc *DuplicateDestination) Execute(ctx context.Context, params DuplicateDestinationParams) (*pipelineservice.Destination, error) {
	group, err := pipelineservice.NormalizeExecutorGroup(params.ExecutorGroup)
	if err != nil {
		return nil, err
	}

	original, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting destination: %w", err)
	}

	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(original.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connector spec: %w", err)
	}

	destID := uuid.New()

	config, err := uc.copyConfig(ctx, original.ID, destID, original.Config, connector.Spec)
	if err != nil {
		return nil, uc.discardSecrets(ctx, destID, err)
	}

	name := params.Name
	if name == "" {
		name = original.Name
	}

	now := time.Now()

	created, err := uc.storage.Destinations().Create(ctx, &pipelineservice.Destination{
		ID:                 destID,
		WorkspaceID:        params.WorkspaceID,
		Name:               name,
		ManagedConnectorID: original.ManagedConnectorID,
		ExecutorGroup:      group,
		Config:             config,
		RuntimeConfig:      original.RuntimeConfig,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return nil, uc.discardSecrets(ctx, destID, fmt.Errorf("creating destination: %w", err))
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "create",
		ResourceType:  "destination",
		ResourceID:    created.ID,
		ResourceLabel: created.Name,
		After: map[string]any{
			"name":                 created.Name,
			"managed_connector_id": created.ManagedConnectorID.String(),
			"config":               auditConfig(created.Config, connector.Spec),
			"executor_group":       created.ExecutorGroup,
			"duplicated_from":      original.ID.String(),
		},
	})

	return created, nil
}

// copyConfig gives every secret of config, referenced or plaintext at a field
// spec marks secret, a new reference owned by id. Referenced secrets must be
// held by the original, originalID.
func (uc *DuplicateDestination) copyConfig(ctx context.Context, originalID, id uuid.UUID, config, spec string) (string, error) {
	copied, err := pipelinesecrets.CopyConfigSecrets(ctx, uc.secrets, "destination", originalID, "destination", id, config)
	if err != nil {
		return "", fmt.Errorf("copying destination config secrets: %w", err)
	}

	encrypted, err := pipelinesecrets.EncryptConfigSecrets(ctx, uc.secrets, "destination", id, copied, spec)
	if err != nil {
		return "", fmt.Errorf("encrypting destination config secrets: %w", err)
	}

	return encrypted, nil
}

// discardSecrets deletes the secrets stored for a copy that was not created
// and returns cause.
func (uc *DuplicateDestination) discardSecrets(ctx context.Context, id uuid.UUID, cause error) error {
	if err := uc.secrets.DeleteByOwner(ctx, "destination", id); err != nil {
		return fmt.Errorf("%w (deleting its secrets: %w)", cause, err)
	}

	return cause
}
