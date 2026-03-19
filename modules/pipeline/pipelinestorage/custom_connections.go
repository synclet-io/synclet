package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// FindDueConnections finds active connections whose next_scheduled_at has passed.
// Excludes connections that already have a scheduled/starting/running job.
// Ordered by next_scheduled_at ASC (earliest-due first).
func (s *ConnectionsStorage) FindDueConnections(ctx context.Context, limit int) ([]pipelinesvc.DueConnection, error) {
	type row struct {
		ConnectionID  uuid.UUID `gorm:"column:id"`
		SourceID      uuid.UUID `gorm:"column:source_id"`
		DestinationID uuid.UUID `gorm:"column:destination_id"`
		Schedule      string    `gorm:"column:schedule"`
		MaxAttempts   int       `gorm:"column:max_attempts"`
	}

	var rows []row
	result := s.DB.WithContext(ctx).
		Raw(`SELECT c.id, c.source_id, c.destination_id, c.schedule, c.max_attempts
		     FROM pipeline.connections c
		     WHERE c.status = ?
		       AND c.next_scheduled_at IS NOT NULL
		       AND c.next_scheduled_at <= NOW()
		       AND NOT EXISTS (
		         SELECT 1 FROM pipeline.jobs j
		         WHERE j.connection_id = c.id
		           AND j.status IN (?, ?, ?)
		       )
		     ORDER BY c.next_scheduled_at ASC
		     LIMIT ?`,
			connectionStatusActive,
			jobStatusScheduled, jobStatusStarting, jobStatusRunning,
			limit).
		Scan(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("finding due connections: %w", result.Error)
	}

	connections := make([]pipelinesvc.DueConnection, len(rows))
	for i, r := range rows {
		connections[i] = pipelinesvc.DueConnection{
			ConnectionID:  r.ConnectionID,
			SourceID:      r.SourceID,
			DestinationID: r.DestinationID,
			Schedule:      r.Schedule,
			MaxAttempts:   r.MaxAttempts,
		}
	}
	return connections, nil
}
