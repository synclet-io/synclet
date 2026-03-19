package pipelinestate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ClearSyncStateParams holds parameters for clearing sync state.
type ClearSyncStateParams struct {
	ConnectionID    uuid.UUID
	StreamNamespace *string // nil = don't filter by namespace
	StreamName      *string // nil = clear all state for connection
}

// ClearSyncState clears state for a connection, or a specific stream within the blob.
type ClearSyncState struct {
	storage pipelineservice.Storage
}

// NewClearSyncState creates a new ClearSyncState use case.
func NewClearSyncState(storage pipelineservice.Storage) *ClearSyncState {
	return &ClearSyncState{storage: storage}
}

// Execute clears sync state. If StreamName is nil, deletes the entire connection state row.
// If StreamName is provided, removes that specific stream from the blob array.
// In both cases, increments generation_id for the affected stream(s) so the destination
// knows to discard old data on the next sync.
func (uc *ClearSyncState) Execute(ctx context.Context, params ClearSyncStateParams) error {
	if params.StreamName == nil {
		// Clear all state for connection — delete the row.
		if err := uc.storage.ConnectionStates().Delete(ctx, &pipelineservice.ConnectionStateFilter{
			ConnectionID: filter.Equals(params.ConnectionID),
		}); err != nil {
			return fmt.Errorf("clearing connection state: %w", err)
		}

		// Increment generation_id for ALL streams of this connection.
		allGens, err := uc.storage.StreamGenerations().Find(ctx, &pipelineservice.StreamGenerationFilter{
			ConnectionID: filter.Equals(params.ConnectionID),
		})
		if err != nil {
			return fmt.Errorf("loading stream generations for connection: %w", err)
		}
		now := time.Now()
		for _, sg := range allGens {
			sg.GenerationID++
			sg.UpdatedAt = now
			if _, err := uc.storage.StreamGenerations().Save(ctx, sg); err != nil {
				return fmt.Errorf("incrementing generation for %s.%s: %w", sg.StreamNamespace, sg.StreamName, err)
			}
		}

		return nil
	}

	// Clear specific stream from blob.
	existing, err := uc.storage.ConnectionStates().First(ctx, &pipelineservice.ConnectionStateFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return nil //nolint:nilerr // not-found is expected, nothing to clear
	}

	var msgs []*protocol.AirbyteStateMessage
	if err := json.Unmarshal([]byte(existing.StateBlob), &msgs); err != nil {
		return fmt.Errorf("parsing state blob: %w", err)
	}

	targetName := *params.StreamName
	targetNS := ""
	if params.StreamNamespace != nil {
		targetNS = *params.StreamNamespace
	}

	// Filter out the matching stream.
	filtered := make([]*protocol.AirbyteStateMessage, 0, len(msgs))
	for _, m := range msgs {
		if m.Type == protocol.StateTypeStream && m.Stream != nil &&
			m.Stream.StreamDescriptor.Name == targetName &&
			m.Stream.StreamDescriptor.Namespace == targetNS {
			continue // skip this stream
		}
		filtered = append(filtered, m)
	}

	data, err := json.Marshal(filtered)
	if err != nil {
		return fmt.Errorf("marshaling filtered state: %w", err)
	}

	existing.StateBlob = string(data)
	existing.UpdatedAt = time.Now()

	if _, err := uc.storage.ConnectionStates().Save(ctx, existing); err != nil {
		return fmt.Errorf("saving filtered state: %w", err)
	}

	// Increment generation_id for the cleared stream.
	if err := uc.incrementStreamGeneration(ctx, params.ConnectionID, targetNS, targetName); err != nil {
		return fmt.Errorf("incrementing stream generation: %w", err)
	}

	return nil
}

// incrementStreamGeneration increments the generation_id for a specific stream.
// If no record exists, creates one with generation_id=1 (first increment from 0).
func (uc *ClearSyncState) incrementStreamGeneration(ctx context.Context, connectionID uuid.UUID, namespace, name string) error {
	existing, err := uc.storage.StreamGenerations().Find(ctx, &pipelineservice.StreamGenerationFilter{
		ConnectionID:    filter.Equals(connectionID),
		StreamNamespace: filter.Equals(namespace),
		StreamName:      filter.Equals(name),
	})
	if err != nil {
		return fmt.Errorf("loading stream generation: %w", err)
	}

	now := time.Now()
	if len(existing) == 0 {
		// No existing record -- create with generation_id=1 (first increment from 0).
		_, err := uc.storage.StreamGenerations().Save(ctx, &pipelineservice.StreamGeneration{
			ConnectionID:    connectionID,
			StreamNamespace: namespace,
			StreamName:      name,
			GenerationID:    1,
			UpdatedAt:       now,
		})
		return err
	}

	existing[0].GenerationID++
	existing[0].UpdatedAt = now
	_, err = uc.storage.StreamGenerations().Save(ctx, existing[0])
	return err
}
