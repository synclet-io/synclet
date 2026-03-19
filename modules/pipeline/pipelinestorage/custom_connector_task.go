package pipelinestorage

import (
	"context"
	"fmt"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ClaimPendingTask atomically claims the oldest pending connector task for the given worker.
// Uses FOR UPDATE SKIP LOCKED to prevent race conditions between multiple executors.
// Returns nil, nil if no pending task is available.
func (s *ConnectorTasksStorage) ClaimPendingTask(ctx context.Context, workerID string) (*pipelinesvc.ConnectorTask, error) {
	var row dbConnectorTask
	result := s.DB.WithContext(ctx).
		Raw(`UPDATE pipeline.connector_tasks
		     SET status = ?, worker_id = ?, updated_at = NOW()
		     WHERE id = (
		         SELECT id FROM pipeline.connector_tasks
		         WHERE status = ?
		         ORDER BY created_at ASC
		         LIMIT 1
		         FOR UPDATE SKIP LOCKED
		     )
		     RETURNING *`,
			connectorTaskStatusRunning, workerID,
			connectorTaskStatusPending).
		Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("claiming pending connector task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return convertConnectorTaskFromDB(&row)
}
