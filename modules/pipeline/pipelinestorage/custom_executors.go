package pipelinestorage

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// Upsert inserts the executor, or updates the row of its InstanceID keeping
// its id and first_seen_at.
func (s *ExecutorsStorage) Upsert(ctx context.Context, executor *pipelinesvc.Executor) error {
	model, err := convertExecutorToDB(executor)
	if err != nil {
		return fmt.Errorf("converting executor to db model: %w", err)
	}

	result := s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "instance_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"type", "version", "capacity", "capabilities", "resource_defaults", "last_seen_at",
			}),
		}).
		Create(model)
	if result.Error != nil {
		return fmt.Errorf("upserting executor: %w", result.Error)
	}

	return nil
}
