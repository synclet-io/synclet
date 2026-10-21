package pipelineconnections

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListConnectionsWithStreams tells which connections have at least one
// configured stream.
type ListConnectionsWithStreams struct {
	storage pipelineservice.Storage
}

// NewListConnectionsWithStreams creates a new ListConnectionsWithStreams use case.
func NewListConnectionsWithStreams(storage pipelineservice.Storage) *ListConnectionsWithStreams {
	return &ListConnectionsWithStreams{storage: storage}
}

// Execute returns the IDs among connectionIDs of the connections that have at
// least one configured stream. The caller has already scoped the connections
// to its workspace.
func (uc *ListConnectionsWithStreams) Execute(ctx context.Context, connectionIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return uc.storage.Connections().WithConfiguredStreams(ctx, connectionIDs)
}
