package pipelinestorage

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// Upsert inserts the stream generation or overwrites generation_id and updated_at
// of the row with the same (connection_id, stream_namespace, stream_name).
// An empty stream namespace is a valid key part.
func (s *StreamGenerationsStorage) Upsert(ctx context.Context, generation *pipelinesvc.StreamGeneration) error {
	model, err := convertStreamGenerationToDB(generation)
	if err != nil {
		return fmt.Errorf("converting stream generation to db model: %w", err)
	}

	result := s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "connection_id"}, {Name: "stream_namespace"}, {Name: "stream_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"generation_id", "updated_at"}),
		}).
		Create(model)
	if result.Error != nil {
		return fmt.Errorf("upserting stream generation: %w", result.Error)
	}

	return nil
}
