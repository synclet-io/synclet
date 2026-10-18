package pipelinecatalog

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const nestedSchema = `{
	"type": "object",
	"properties": {
		"id": {"type": "integer"},
		"name": {"type": "string"},
		"updated_at": {"type": "string"},
		"address": {"type": "object", "properties": {"city": {"type": "string"}}}
	}
}`

func fields(paths ...[]string) []protocol.SelectedField {
	out := make([]protocol.SelectedField, len(paths))
	for i, p := range paths {
		out[i] = protocol.SelectedField{FieldPath: p}
	}

	return out
}

func TestValidateSelectedFields(t *testing.T) {
	t.Run("accepts top-level and nested paths", func(t *testing.T) {
		err := ValidateSelectedFields(fields([]string{"id"}, []string{"address", "city"}), json.RawMessage(nestedSchema))
		require.NoError(t, err)
	})

	t.Run("rejects unknown path", func(t *testing.T) {
		err := ValidateSelectedFields(fields([]string{"address", "zip"}), json.RawMessage(nestedSchema))
		require.ErrorContains(t, err, "[address zip] not found")
	})

	t.Run("rejects empty path", func(t *testing.T) {
		err := ValidateSelectedFields(fields([]string{}), json.RawMessage(nestedSchema))
		require.ErrorIs(t, err, pipelineservice.ErrEmptyFieldPath)
	})

	t.Run("rejects invalid schema", func(t *testing.T) {
		err := ValidateSelectedFields(fields([]string{"id"}), json.RawMessage(`nope`))
		require.ErrorContains(t, err, "parsing json schema")
	})
}

func TestForceIncludeFields(t *testing.T) {
	t.Run("adds missing cursor and primary key paths", func(t *testing.T) {
		selected := fields([]string{"name"})

		got := ForceIncludeFields(selected, []string{"updated_at"}, [][]string{{"id"}, {"address", "city"}})

		assert.Equal(t, fields([]string{"name"}, []string{"updated_at"}, []string{"id"}, []string{"address", "city"}), got)
		assert.Len(t, selected, 1, "input slice must not be mutated")
	})

	t.Run("does not duplicate already selected paths", func(t *testing.T) {
		selected := fields([]string{"id"}, []string{"updated_at"})

		got := ForceIncludeFields(selected, []string{"updated_at"}, [][]string{{"id"}})

		assert.Equal(t, selected, got)
	})

	t.Run("ignores empty cursor and primary key paths", func(t *testing.T) {
		selected := fields([]string{"name"})

		got := ForceIncludeFields(selected, nil, [][]string{{}})

		assert.Equal(t, selected, got)
	})
}

func TestFilterRecordData(t *testing.T) {
	t.Run("keeps only selected top-level fields", func(t *testing.T) {
		got, err := FilterRecordData(
			json.RawMessage(`{"id":1,"name":"a","address":{"city":"x","zip":"y"},"secret":"s"}`),
			fields([]string{"id"}, []string{"address", "city"}, []string{}),
		)
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":1,"address":{"city":"x","zip":"y"}}`, string(got))
	})

	t.Run("no selected fields yields empty object", func(t *testing.T) {
		got, err := FilterRecordData(json.RawMessage(`{"id":1}`), nil)
		require.NoError(t, err)
		assert.JSONEq(t, `{}`, string(got))
	})

	t.Run("rejects non-object record", func(t *testing.T) {
		_, err := FilterRecordData(json.RawMessage(`[1,2]`), fields([]string{"id"}))
		require.ErrorContains(t, err, "parsing record data")
	})
}

func TestBuildDestinationCatalog(t *testing.T) {
	t.Run("filters schema only for streams with selected fields", func(t *testing.T) {
		catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:         protocol.AirbyteStream{Name: "users", JSONSchema: json.RawMessage(nestedSchema)},
				SelectedFields: fields([]string{"id"}, []string{"address", "city"}),
			},
			{
				Stream: protocol.AirbyteStream{Name: "orders", JSONSchema: json.RawMessage(nestedSchema)},
			},
		}}

		dest, err := BuildDestinationCatalog(catalog)
		require.NoError(t, err)
		require.Len(t, dest.Streams, 2)

		assert.JSONEq(t, `{
			"type": "object",
			"properties": {
				"id": {"type": "integer"},
				"address": {"type": "object", "properties": {"city": {"type": "string"}}}
			}
		}`, string(dest.Streams[0].Stream.JSONSchema))
		assert.JSONEq(t, nestedSchema, string(dest.Streams[1].Stream.JSONSchema))
		assert.JSONEq(t, nestedSchema, string(catalog.Streams[0].Stream.JSONSchema), "source catalog must not be mutated")
	})

	t.Run("schema without properties passes through", func(t *testing.T) {
		catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
			Stream:         protocol.AirbyteStream{Name: "users", JSONSchema: json.RawMessage(`{"type":"object"}`)},
			SelectedFields: fields([]string{"id"}),
		}}}

		dest, err := BuildDestinationCatalog(catalog)
		require.NoError(t, err)
		assert.JSONEq(t, `{"type":"object"}`, string(dest.Streams[0].Stream.JSONSchema))
	})

	t.Run("invalid schema fails with stream name", func(t *testing.T) {
		catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
			Stream:         protocol.AirbyteStream{Name: "users", JSONSchema: json.RawMessage(`nope`)},
			SelectedFields: fields([]string{"id"}),
		}}}

		_, err := BuildDestinationCatalog(catalog)
		require.ErrorContains(t, err, `stream "users"`)
	})

	t.Run("invalid properties fails", func(t *testing.T) {
		_, err := filterSchemaBySelectedFields(json.RawMessage(`{"properties":[1]}`), fields([]string{"id"}))
		require.ErrorContains(t, err, "parsing properties")
	})
}
