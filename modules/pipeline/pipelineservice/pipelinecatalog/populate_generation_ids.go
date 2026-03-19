package pipelinecatalog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// PopulateGenerationIDsParams holds parameters for populating generation IDs.
type PopulateGenerationIDsParams struct {
	ConnectionID uuid.UUID
	Catalog      *protocol.ConfiguredAirbyteCatalog
	SyncID       int64
}

// PopulateGenerationIDs loads existing per-stream generation counters from the database,
// resolves the correct generation_id and minimum_generation_id for each stream in the
// catalog, and persists updated counters back to the database.
type PopulateGenerationIDs struct {
	storage pipelineservice.Storage
}

// NewPopulateGenerationIDs creates a new PopulateGenerationIDs use case.
func NewPopulateGenerationIDs(storage pipelineservice.Storage) *PopulateGenerationIDs {
	return &PopulateGenerationIDs{storage: storage}
}

// Execute loads existing generation records, resolves generation IDs for each stream,
// and saves the updated records.
func (uc *PopulateGenerationIDs) Execute(ctx context.Context, params PopulateGenerationIDsParams) error {
	existing, err := uc.storage.StreamGenerations().Find(ctx, &pipelineservice.StreamGenerationFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return fmt.Errorf("loading stream generations: %w", err)
	}

	toSave := ResolveStreamGenerations(params.Catalog, existing, params.ConnectionID, params.SyncID)

	for _, sg := range toSave {
		if _, err := uc.storage.StreamGenerations().Save(ctx, sg); err != nil {
			return fmt.Errorf("saving stream generation for %s.%s: %w", sg.StreamNamespace, sg.StreamName, err)
		}
	}

	return nil
}

// ResolveStreamGenerations is a pure function that resolves generation_id and minimum_generation_id
// for each stream in the catalog. It also sets SyncID on each stream.
//
// Rules:
//   - full_refresh streams: generation_id is incremented each sync
//   - incremental streams: generation_id stays unchanged
//   - overwrite destination: minimum_generation_id = generation_id (clear old data)
//   - append/append_dedup: minimum_generation_id = 0 (keep all data)
//
// Returns the list of StreamGeneration records that need to be saved (upserted).
func ResolveStreamGenerations(
	catalog *protocol.ConfiguredAirbyteCatalog,
	existingGens []*pipelineservice.StreamGeneration,
	connectionID uuid.UUID,
	syncID int64,
) []*pipelineservice.StreamGeneration {
	genMap := make(map[string]*pipelineservice.StreamGeneration, len(existingGens))
	for _, sg := range existingGens {
		genMap[streamKey(sg.StreamNamespace, sg.StreamName)] = sg
	}

	var toSave []*pipelineservice.StreamGeneration
	now := time.Now()

	for i, s := range catalog.Streams {
		catalog.Streams[i].SyncID = syncID

		key := streamKey(s.Stream.Namespace, s.Stream.Name)
		genID := int64(0)
		if existing := genMap[key]; existing != nil {
			genID = existing.GenerationID
		}

		// Increment generation for full_refresh syncs.
		if s.SyncMode == protocol.SyncModeFullRefresh {
			genID++
		}

		catalog.Streams[i].GenerationID = genID

		// Set minimum_generation_id based on destination sync mode.
		switch s.DestinationSyncMode {
		case protocol.DestinationSyncModeOverwrite:
			catalog.Streams[i].MinimumGenerationID = genID
		default:
			catalog.Streams[i].MinimumGenerationID = 0
		}

		toSave = append(toSave, &pipelineservice.StreamGeneration{
			ConnectionID:    connectionID,
			StreamNamespace: s.Stream.Namespace,
			StreamName:      s.Stream.Name,
			GenerationID:    genID,
			UpdatedAt:       now,
		})
	}

	return toSave
}
