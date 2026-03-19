package pipelinecatalog

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestResolveStreamGenerations_FullRefreshNewStream(t *testing.T) {
	// No existing generation -> full_refresh increments from 0 to 1
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "users", Namespace: "public"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			},
		},
	}
	connID := uuid.New()
	toSave := ResolveStreamGenerations(catalog, nil, connID, 5)

	assert.Equal(t, int64(1), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(1), catalog.Streams[0].MinimumGenerationID)
	assert.Equal(t, int64(5), catalog.Streams[0].SyncID)
	require.Len(t, toSave, 1)
	assert.Equal(t, int64(1), toSave[0].GenerationID)
	assert.Equal(t, connID, toSave[0].ConnectionID)
	assert.Equal(t, "public", toSave[0].StreamNamespace)
	assert.Equal(t, "users", toSave[0].StreamName)
}

func TestResolveStreamGenerations_FullRefreshExistingGeneration(t *testing.T) {
	// Existing generation=3 -> full_refresh increments to 4
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "users", Namespace: "public"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			},
		},
	}
	connID := uuid.New()
	existing := []*pipelineservice.StreamGeneration{
		{
			ConnectionID:    connID,
			StreamNamespace: "public",
			StreamName:      "users",
			GenerationID:    3,
		},
	}
	toSave := ResolveStreamGenerations(catalog, existing, connID, 10)

	assert.Equal(t, int64(4), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(4), catalog.Streams[0].MinimumGenerationID)
	assert.Equal(t, int64(10), catalog.Streams[0].SyncID)
	require.Len(t, toSave, 1)
	assert.Equal(t, int64(4), toSave[0].GenerationID)
}

func TestResolveStreamGenerations_IncrementalNewStream(t *testing.T) {
	// No existing generation -> incremental stays at 0
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "events", Namespace: "public"},
				SyncMode:            protocol.SyncModeIncremental,
				DestinationSyncMode: protocol.DestinationSyncModeAppend,
			},
		},
	}
	connID := uuid.New()
	toSave := ResolveStreamGenerations(catalog, nil, connID, 7)

	assert.Equal(t, int64(0), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(0), catalog.Streams[0].MinimumGenerationID)
	assert.Equal(t, int64(7), catalog.Streams[0].SyncID)
	require.Len(t, toSave, 1)
	assert.Equal(t, int64(0), toSave[0].GenerationID)
}

func TestResolveStreamGenerations_IncrementalExistingGeneration(t *testing.T) {
	// Existing generation=3 -> incremental stays at 3 (no increment)
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "events", Namespace: "public"},
				SyncMode:            protocol.SyncModeIncremental,
				DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
			},
		},
	}
	connID := uuid.New()
	existing := []*pipelineservice.StreamGeneration{
		{
			ConnectionID:    connID,
			StreamNamespace: "public",
			StreamName:      "events",
			GenerationID:    3,
		},
	}
	toSave := ResolveStreamGenerations(catalog, existing, connID, 12)

	assert.Equal(t, int64(3), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(0), catalog.Streams[0].MinimumGenerationID)
	assert.Equal(t, int64(12), catalog.Streams[0].SyncID)
	require.Len(t, toSave, 1)
	assert.Equal(t, int64(3), toSave[0].GenerationID)
}

func TestResolveStreamGenerations_OverwriteMinGenEqualsGenID(t *testing.T) {
	// overwrite dest mode -> minimum_generation_id = generation_id
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "users"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			},
		},
	}
	connID := uuid.New()
	existing := []*pipelineservice.StreamGeneration{
		{ConnectionID: connID, StreamName: "users", GenerationID: 5},
	}
	ResolveStreamGenerations(catalog, existing, connID, 1)

	assert.Equal(t, int64(6), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(6), catalog.Streams[0].MinimumGenerationID)
}

func TestResolveStreamGenerations_AppendMinGenIsZero(t *testing.T) {
	// append dest mode -> minimum_generation_id = 0
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "logs"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeAppend,
			},
		},
	}
	connID := uuid.New()
	ResolveStreamGenerations(catalog, nil, connID, 1)

	assert.Equal(t, int64(1), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(0), catalog.Streams[0].MinimumGenerationID)
}

func TestResolveStreamGenerations_AppendDedupMinGenIsZero(t *testing.T) {
	// append_dedup dest mode -> minimum_generation_id = 0
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "orders"},
				SyncMode:            protocol.SyncModeIncremental,
				DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
			},
		},
	}
	connID := uuid.New()
	existing := []*pipelineservice.StreamGeneration{
		{ConnectionID: connID, StreamName: "orders", GenerationID: 2},
	}
	ResolveStreamGenerations(catalog, existing, connID, 1)

	assert.Equal(t, int64(2), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(0), catalog.Streams[0].MinimumGenerationID)
}

func TestResolveStreamGenerations_MixedStreams(t *testing.T) {
	// Multiple streams with mixed sync modes
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "users", Namespace: "public"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			},
			{
				Stream:              protocol.AirbyteStream{Name: "events", Namespace: "public"},
				SyncMode:            protocol.SyncModeIncremental,
				DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
			},
			{
				Stream:              protocol.AirbyteStream{Name: "logs"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeAppend,
			},
		},
	}
	connID := uuid.New()
	existing := []*pipelineservice.StreamGeneration{
		{ConnectionID: connID, StreamNamespace: "public", StreamName: "users", GenerationID: 2},
		{ConnectionID: connID, StreamNamespace: "public", StreamName: "events", GenerationID: 5},
	}
	toSave := ResolveStreamGenerations(catalog, existing, connID, 42)

	require.Len(t, toSave, 3)

	// users: full_refresh + overwrite -> gen 3, min_gen 3
	assert.Equal(t, int64(3), catalog.Streams[0].GenerationID)
	assert.Equal(t, int64(3), catalog.Streams[0].MinimumGenerationID)
	assert.Equal(t, int64(42), catalog.Streams[0].SyncID)

	// events: incremental + append_dedup -> gen 5 (no change), min_gen 0
	assert.Equal(t, int64(5), catalog.Streams[1].GenerationID)
	assert.Equal(t, int64(0), catalog.Streams[1].MinimumGenerationID)
	assert.Equal(t, int64(42), catalog.Streams[1].SyncID)

	// logs: full_refresh + append -> gen 1 (new), min_gen 0
	assert.Equal(t, int64(1), catalog.Streams[2].GenerationID)
	assert.Equal(t, int64(0), catalog.Streams[2].MinimumGenerationID)
	assert.Equal(t, int64(42), catalog.Streams[2].SyncID)
}

func TestResolveStreamGenerations_SyncIDSetOnAllStreams(t *testing.T) {
	catalog := &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{Stream: protocol.AirbyteStream{Name: "a"}, SyncMode: protocol.SyncModeIncremental, DestinationSyncMode: protocol.DestinationSyncModeAppend},
			{Stream: protocol.AirbyteStream{Name: "b"}, SyncMode: protocol.SyncModeFullRefresh, DestinationSyncMode: protocol.DestinationSyncModeOverwrite},
		},
	}
	connID := uuid.New()
	ResolveStreamGenerations(catalog, nil, connID, 99)

	assert.Equal(t, int64(99), catalog.Streams[0].SyncID)
	assert.Equal(t, int64(99), catalog.Streams[1].SyncID)
}
