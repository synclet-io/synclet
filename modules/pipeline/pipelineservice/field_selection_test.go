package pipelineservice

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		require.ErrorIs(t, err, ErrEmptyFieldPath)
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

const ordersSchema = `{
	"type": "object",
	"properties": {
		"cust_id": {"type": "integer"},
		"amount": {"type": "number"}
	}
}`

func TestBuildDestinationCatalogMappers(t *testing.T) {
	t.Run("rename rewrites the schema and the primary key", func(t *testing.T) {
		catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
			Stream:              protocol.AirbyteStream{Name: "orders", JSONSchema: json.RawMessage(ordersSchema)},
			DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
			PrimaryKey:          [][]string{{"cust_id"}},
			Mappers:             json.RawMessage(`[{"rename":{"cust_id":"customer_id"}}]`),
		}}}

		dest, err := BuildDestinationCatalog(catalog)
		require.NoError(t, err)

		assert.JSONEq(t, `{
			"type": "object",
			"properties": {
				"customer_id": {"type": "integer"},
				"amount": {"type": "number"}
			}
		}`, string(dest.Streams[0].Stream.JSONSchema))
		assert.Equal(t, [][]string{{"customer_id"}}, dest.Streams[0].PrimaryKey)
		assert.Nil(t, dest.Streams[0].Mappers)
		assert.NotContains(t, marshalCatalog(t, dest), "mappers")

		assert.Equal(t, [][]string{{"cust_id"}}, catalog.Streams[0].PrimaryKey, "configured catalog must not be mutated")
		assert.JSONEq(t, ordersSchema, string(catalog.Streams[0].Stream.JSONSchema))
	})

	t.Run("set on an unselected field's value", func(t *testing.T) {
		catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
			Stream:              protocol.AirbyteStream{Name: "orders", JSONSchema: json.RawMessage(ordersSchema)},
			DestinationSyncMode: protocol.DestinationSyncModeAppend,
			SelectedFields:      fields([]string{"cust_id"}),
			Mappers:             json.RawMessage(`[{"set":{"total_cents":"int(r.amount * 100.0)"}}]`),
		}}}

		dest, err := BuildDestinationCatalog(catalog)
		require.NoError(t, err)

		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		require.NoError(t, json.Unmarshal(dest.Streams[0].Stream.JSONSchema, &schema))
		assert.Contains(t, schema.Properties, "total_cents")
		assert.Contains(t, schema.Properties, "cust_id")
		assert.NotContains(t, schema.Properties, "amount")
	})

	t.Run("mappers that do not compile fail with the stream and step", func(t *testing.T) {
		catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
			Stream:  protocol.AirbyteStream{Name: "orders", JSONSchema: json.RawMessage(ordersSchema)},
			Mappers: json.RawMessage(`[{"set":{"x":"r.nope"}}]`),
		}}}

		_, err := BuildDestinationCatalog(catalog)
		require.ErrorContains(t, err, `stream "orders"`)
		require.ErrorContains(t, err, "step 0")
	})
}

func TestBuildSourceCatalog(t *testing.T) {
	catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{
		{
			Stream:              protocol.AirbyteStream{Name: "orders", JSONSchema: json.RawMessage(ordersSchema)},
			DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
			PrimaryKey:          [][]string{{"cust_id"}},
			SelectedFields:      fields([]string{"cust_id"}),
			Mappers:             json.RawMessage(`[{"rename":{"cust_id":"customer_id"}}]`),
		},
		{
			Stream:         protocol.AirbyteStream{Name: "users", JSONSchema: json.RawMessage(nestedSchema)},
			SelectedFields: fields([]string{"id"}),
		},
	}}

	src := BuildSourceCatalog(catalog)

	require.Len(t, src.Streams, 2)
	assert.JSONEq(t, ordersSchema, string(src.Streams[0].Stream.JSONSchema))
	assert.Equal(t, [][]string{{"cust_id"}}, src.Streams[0].PrimaryKey)
	assert.Equal(t, fields([]string{"cust_id"}), src.Streams[0].SelectedFields)
	assert.Nil(t, src.Streams[0].Mappers)
	assert.NotContains(t, marshalCatalog(t, src), "mappers")
	assert.JSONEq(t, nestedSchema, string(src.Streams[1].Stream.JSONSchema))
	assert.NotNil(t, catalog.Streams[0].Mappers, "configured catalog must not be mutated")
}

func marshalCatalog(t *testing.T, catalog *protocol.ConfiguredAirbyteCatalog) string {
	t.Helper()

	raw, err := json.Marshal(catalog)
	require.NoError(t, err)

	return string(raw)
}

func TestBuildDestinationCatalogRenamedCursor(t *testing.T) {
	catalog := &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{{
		Stream:              protocol.AirbyteStream{Name: "orders", JSONSchema: json.RawMessage(ordersSchema)},
		SyncMode:            protocol.SyncModeIncremental,
		DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
		PrimaryKey:          [][]string{{"cust_id"}},
		CursorField:         []string{"amount"},
		Mappers:             json.RawMessage(`[{"rename":{"amount":"total"}},{"rename":{"total":"order_total","cust_id":"customer_id"}}]`),
	}}}

	dest, err := BuildDestinationCatalog(catalog)
	require.NoError(t, err)
	assert.Equal(t, []string{"order_total"}, dest.Streams[0].CursorField)
	assert.Equal(t, [][]string{{"customer_id"}}, dest.Streams[0].PrimaryKey)
	assert.Contains(t, string(dest.Streams[0].Stream.JSONSchema), `"order_total"`)

	assert.Equal(t, []string{"amount"}, catalog.Streams[0].CursorField, "configured catalog must not be mutated")
	assert.Equal(t, []string{"amount"}, BuildSourceCatalog(catalog).Streams[0].CursorField)
}
