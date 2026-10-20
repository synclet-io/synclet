package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// FindDueConnections finds active connections whose next_scheduled_at has passed.
// Excludes connections that already have a scheduled/starting/running job and
// connections whose workspace no longer exists.
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
		     JOIN workspace.workspaces w ON w.id = c.workspace_id
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
	for i, row := range rows {
		connections[i] = pipelinesvc.DueConnection{
			ConnectionID:  row.ConnectionID,
			SourceID:      row.SourceID,
			DestinationID: row.DestinationID,
			Schedule:      row.Schedule,
			MaxAttempts:   row.MaxAttempts,
		}
	}

	return connections, nil
}

// ListTagCounts returns every tag used by the workspace's connections with the
// number of connections carrying it, ordered by tag.
func (s *ConnectionsStorage) ListTagCounts(ctx context.Context, workspaceID uuid.UUID) ([]pipelinesvc.TagCount, error) {
	type row struct {
		Tag             string `gorm:"column:tag"`
		ConnectionCount int    `gorm:"column:connection_count"`
	}

	var rows []row

	result := s.DB.WithContext(ctx).
		Raw(`SELECT tag, COUNT(*) AS connection_count
		     FROM pipeline.connections c, unnest(c.tags) AS tag
		     WHERE c.workspace_id = ?
		     GROUP BY tag
		     ORDER BY tag`,
			workspaceID).
		Scan(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("listing connection tag counts: %w", result.Error)
	}

	counts := make([]pipelinesvc.TagCount, len(rows))
	for i, row := range rows {
		counts[i] = pipelinesvc.TagCount{Tag: row.Tag, ConnectionCount: row.ConnectionCount}
	}

	return counts, nil
}
