package pipelinestate

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// ListStreamStatesParams holds parameters for listing stream states with workspace verification.
type ListStreamStatesParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// ListStreamStates verifies workspace ownership and returns stream states.
type ListStreamStates struct {
	getConnection  *pipelineconnections.GetConnection
	listSyncStates *ListSyncStates
}

// NewListStreamStates creates a new ListStreamStates use case.
func NewListStreamStates(getConnection *pipelineconnections.GetConnection, listSyncStates *ListSyncStates) *ListStreamStates {
	return &ListStreamStates{
		getConnection:  getConnection,
		listSyncStates: listSyncStates,
	}
}

// Execute verifies the connection belongs to the workspace and lists stream states.
func (uc *ListStreamStates) Execute(ctx context.Context, params ListStreamStatesParams) (*ListSyncStatesResult, error) {
	if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	result, err := uc.listSyncStates.Execute(ctx, ListSyncStatesParams{
		ConnectionID: params.ConnectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing stream states: %w", err)
	}

	return result, nil
}
