package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// UsedManagedConnectorIDs returns those of ids that a source or destination of
// the workspace is built on.
func (s *ManagedConnectorsStorage) UsedManagedConnectorIDs(ctx context.Context, workspaceID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var used []uuid.UUID

	result := s.DB.WithContext(ctx).
		Raw(`SELECT managed_connector_id FROM pipeline.sources
		     WHERE workspace_id = ? AND managed_connector_id IN ?
		     UNION
		     SELECT managed_connector_id FROM pipeline.destinations
		     WHERE workspace_id = ? AND managed_connector_id IN ?`,
			workspaceID, ids, workspaceID, ids).
		Scan(&used)
	if result.Error != nil {
		return nil, fmt.Errorf("finding used managed connectors: %w", result.Error)
	}

	return used, nil
}
