package pipelinesources

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
)

// DuplicateSource copies a source into another executor group.
type DuplicateSource struct {
	storage pipelineservice.Storage
	secrets pipelineservice.SecretsProvider
	audit   pipelineservice.AuditRecorder
}

// NewDuplicateSource creates a new DuplicateSource use case.
func NewDuplicateSource(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, audit pipelineservice.AuditRecorder) *DuplicateSource {
	return &DuplicateSource{storage: storage, secrets: secrets, audit: audit}
}

// DuplicateSourceParams holds parameters for duplicating a source.
type DuplicateSourceParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	// ExecutorGroup is the group of the copy; empty means default.
	ExecutorGroup string
	// Name is the name of the copy; empty keeps the original's name.
	Name string
}

// Execute creates a source with the original's managed connector, config and
// runtime config in params.ExecutorGroup. The copy's secrets are stored anew
// under its own id; connections and their state stay with the original.
func (uc *DuplicateSource) Execute(ctx context.Context, params DuplicateSourceParams) (*pipelineservice.Source, error) {
	group, err := pipelineservice.NormalizeExecutorGroup(params.ExecutorGroup)
	if err != nil {
		return nil, err
	}

	original, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting source: %w", err)
	}

	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(original.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connector spec: %w", err)
	}

	srcID := uuid.New()

	config, err := uc.copyConfig(ctx, original.ID, srcID, original.Config, connector.Spec)
	if err != nil {
		return nil, uc.discardSecrets(ctx, srcID, err)
	}

	name := params.Name
	if name == "" {
		name = original.Name
	}

	now := time.Now()

	created, err := uc.storage.Sources().Create(ctx, &pipelineservice.Source{
		ID:                 srcID,
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
		return nil, uc.discardSecrets(ctx, srcID, fmt.Errorf("creating source: %w", err))
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "create",
		ResourceType:  "source",
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
func (uc *DuplicateSource) copyConfig(ctx context.Context, originalID, id uuid.UUID, config, spec string) (string, error) {
	copied, err := pipelinesecrets.CopyConfigSecrets(ctx, uc.secrets, "source", originalID, "source", id, config)
	if err != nil {
		return "", fmt.Errorf("copying source config secrets: %w", err)
	}

	encrypted, err := pipelinesecrets.EncryptConfigSecrets(ctx, uc.secrets, "source", id, copied, spec)
	if err != nil {
		return "", fmt.Errorf("encrypting source config secrets: %w", err)
	}

	return encrypted, nil
}

// discardSecrets deletes the secrets stored for a copy that was not created
// and returns cause.
func (uc *DuplicateSource) discardSecrets(ctx context.Context, id uuid.UUID, cause error) error {
	if err := uc.secrets.DeleteByOwner(ctx, "source", id); err != nil {
		return fmt.Errorf("%w (deleting its secrets: %w)", cause, err)
	}

	return cause
}
