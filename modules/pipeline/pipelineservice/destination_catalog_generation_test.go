package pipelineservice_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// The destination catalog carries the generation and sync ids resolved from the
// stored stream generations, through field selection and namespace/prefix rewriting.
func TestDestinationCatalog_CarriesGenerationFields(t *testing.T) {
	connectionID := uuid.New()
	schema := json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}}}`)
	source := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{
		{
			Stream:              protocol.AirbyteStream{Name: "users", Namespace: "public", JSONSchema: schema},
			SyncMode:            protocol.SyncModeFullRefresh,
			DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			SelectedFields:      []protocol.SelectedField{{FieldPath: []string{"id"}}},
		},
		{
			Stream:              protocol.AirbyteStream{Name: "events", Namespace: "public", JSONSchema: schema},
			SyncMode:            protocol.SyncModeIncremental,
			DestinationSyncMode: protocol.DestinationSyncModeAppend,
		},
		{
			Stream:              protocol.AirbyteStream{Name: "orders", Namespace: "public", JSONSchema: schema},
			SyncMode:            protocol.SyncModeIncremental,
			DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
		},
	}}
	stored := []*pipelineservice.StreamGeneration{
		{ConnectionID: connectionID, StreamNamespace: "public", StreamName: "users", GenerationID: 4},
		{ConnectionID: connectionID, StreamNamespace: "public", StreamName: "events", GenerationID: 2},
	}

	pipelinecatalog.ResolveStreamGenerations(source, stored, connectionID, 7)

	bundleCatalog, err := json.Marshal(source)
	require.NoError(t, err)

	var claimed protocol.ConfiguredAirbyteCatalog
	require.NoError(t, json.Unmarshal(bundleCatalog, &claimed))

	dest, err := pipelineservice.BuildDestinationCatalog(&claimed)
	require.NoError(t, err)

	format, prefix := "raw_${SOURCE_NAMESPACE}", "sl_"
	pipelineservice.ApplyNamespaceAndPrefix(dest, pipelineservice.NamespaceDefinitionCustom, &format, &prefix)

	destJSON, err := json.Marshal(dest)
	require.NoError(t, err)

	var sent struct {
		Streams []map[string]any `json:"streams"`
	}
	decoder := json.NewDecoder(bytes.NewReader(destJSON))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&sent))
	require.Len(t, sent.Streams, 3)

	want := []struct {
		name                string
		generation, minimum json.Number
	}{
		{name: "sl_users", generation: "5", minimum: "5"},
		{name: "sl_events", generation: "2", minimum: "0"},
		{name: "sl_orders", generation: "0", minimum: "0"},
	}
	for i, w := range want {
		stream := sent.Streams[i]
		assert.Equal(t, w.name, stream["stream"].(map[string]any)["name"])
		assert.Equal(t, "raw_public", stream["stream"].(map[string]any)["namespace"])

		for _, key := range []string{"generation_id", "minimum_generation_id", "sync_id"} {
			assert.Contains(t, stream, key, w.name)
		}

		assert.Equal(t, w.generation, stream["generation_id"], w.name)
		assert.Equal(t, w.minimum, stream["minimum_generation_id"], w.name)
		assert.Equal(t, json.Number("7"), stream["sync_id"], w.name)
	}
}
