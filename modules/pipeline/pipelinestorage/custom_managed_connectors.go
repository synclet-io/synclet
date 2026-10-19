package pipelinestorage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
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

// SetStarred writes only the star flag and updated_at of the workspace's
// managed connector. It returns ErrManagedConnectorNotFound when no row matches.
func (s *ManagedConnectorsStorage) SetStarred(ctx context.Context, workspaceID, id uuid.UUID, starred bool) error {
	result := s.DB.WithContext(ctx).
		Exec(`UPDATE pipeline.managed_connectors SET starred = ?, updated_at = ? WHERE id = ? AND workspace_id = ?`,
			starred, time.Now(), id, workspaceID)
	if result.Error != nil {
		return fmt.Errorf("setting managed connector star: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pipelineservice.ErrManagedConnectorNotFound
	}

	return nil
}
