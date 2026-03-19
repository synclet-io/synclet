package pipelineconnections

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CreateConnectionParams holds parameters for creating a connection.
type CreateConnectionParams struct {
	WorkspaceID           uuid.UUID
	Name                  string
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	Schedule              *string
	SchemaChangePolicy    pipelineservice.SchemaChangePolicy
	MaxAttempts           int
	NamespaceDefinition   pipelineservice.NamespaceDefinition
	CustomNamespaceFormat *string
	StreamPrefix          *string
}

// CreateConnection creates a new connection between a source and a destination.
// It validates that the source and destination exist within the workspace using
// direct storage queries, eliminating the need for SourceValidator and
// DestinationValidator adapter interfaces.
type CreateConnection struct {
	storage pipelineservice.Storage
}

// NewCreateConnection creates a new CreateConnection use case.
func NewCreateConnection(storage pipelineservice.Storage) *CreateConnection {
	return &CreateConnection{storage: storage}
}

// Execute validates source/destination existence and creates the connection.
func (uc *CreateConnection) Execute(ctx context.Context, params CreateConnectionParams) (*pipelineservice.Connection, error) {
	// Validate source exists in the workspace — direct storage call, no adapter needed.
	_, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.SourceID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("validating source: %w", err)
	}

	// Validate destination exists in the workspace — direct storage call, no adapter needed.
	_, err = uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(params.DestinationID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("validating destination: %w", err)
	}

	if params.Schedule != nil && *params.Schedule != "" {
		if _, err := pipelineservice.CronParser.Parse(*params.Schedule); err != nil {
			return nil, fmt.Errorf("invalid cron expression %q: %w", *params.Schedule, err)
		}
	}

	now := time.Now()

	policy := params.SchemaChangePolicy
	if !policy.IsValid() {
		policy = pipelineservice.SchemaChangePolicyPause
	}

	maxAttempts := params.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	nsDef := params.NamespaceDefinition
	if !nsDef.IsValid() {
		nsDef = pipelineservice.NamespaceDefinitionSource
	}

	conn := &pipelineservice.Connection{
		ID:                    uuid.New(),
		WorkspaceID:           params.WorkspaceID,
		Name:                  params.Name,
		Status:                pipelineservice.ConnectionStatusActive,
		SourceID:              params.SourceID,
		DestinationID:         params.DestinationID,
		Schedule:              params.Schedule,
		SchemaChangePolicy:    policy,
		MaxAttempts:           maxAttempts,
		NamespaceDefinition:   nsDef,
		CustomNamespaceFormat: params.CustomNamespaceFormat,
		StreamPrefix:          params.StreamPrefix,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	pipelineservice.RecomputeNextScheduledAt(conn, now)

	created, err := uc.storage.Connections().Create(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("creating connection: %w", err)
	}

	return created, nil
}
