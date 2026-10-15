package pipelinecatalog

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/pkg/protocol"
)

func stream(namespace, name, schema string) protocol.AirbyteStream {
	return protocol.AirbyteStream{Namespace: namespace, Name: name, JSONSchema: json.RawMessage(schema)}
}

func TestComputeSchemaDiff(t *testing.T) {
	t.Run("identical catalogs produce no changes", func(t *testing.T) {
		catalog := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
			stream("public", "users", `{"properties":{"id":{"type":"integer"}}}`),
		}}

		assert.Empty(t, ComputeSchemaDiff(catalog, catalog))
	})

	t.Run("detects stream and column changes in deterministic order", func(t *testing.T) {
		old := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
			stream("public", "users", `{"properties":{
				"id":{"type":"integer"},
				"email":{"type":["string","null"]},
				"legacy":{"type":"string"},
				"age":{"type":"integer"}
			}}`),
			stream("public", "orders", `{}`),
		}}
		updated := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
			stream("public", "users", `{"properties":{
				"id":{"type":"integer"},
				"email":{"type":"string"},
				"age":{"type":["null","string"]},
				"created_at":{"type":"string"}
			}}`),
			stream("public", "payments", `{}`),
		}}

		assert.Equal(t, []SchemaChange{
			{Type: StreamRemoved, StreamName: "orders", Namespace: "public"},
			{Type: StreamAdded, StreamName: "payments", Namespace: "public"},
			{Type: ColumnAdded, StreamName: "users", Namespace: "public", ColumnName: "created_at"},
			{Type: ColumnRemoved, StreamName: "users", Namespace: "public", ColumnName: "legacy"},
			{Type: ColumnTypeChanged, StreamName: "users", Namespace: "public", ColumnName: "age", OldType: "integer", NewType: "string"},
		}, ComputeSchemaDiff(old, updated))
	})

	t.Run("nil catalogs are treated as empty", func(t *testing.T) {
		catalog := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{stream("", "users", "")}}

		assert.Empty(t, ComputeSchemaDiff(nil, nil))
		assert.Equal(t, []SchemaChange{{Type: StreamAdded, StreamName: "users"}}, ComputeSchemaDiff(nil, catalog))
		assert.Equal(t, []SchemaChange{{Type: StreamRemoved, StreamName: "users"}}, ComputeSchemaDiff(catalog, nil))
	})

	t.Run("same name in different namespaces are distinct streams", func(t *testing.T) {
		old := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{stream("a", "users", "")}}
		updated := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{stream("b", "users", "")}}

		assert.Equal(t, []SchemaChange{
			{Type: StreamRemoved, StreamName: "users", Namespace: "a"},
			{Type: StreamAdded, StreamName: "users", Namespace: "b"},
		}, ComputeSchemaDiff(old, updated))
	})

	t.Run("unparseable schemas yield no column changes", func(t *testing.T) {
		old := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{stream("", "users", `not json`)}}
		updated := &protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{stream("", "users", `[]`)}}

		assert.Empty(t, ComputeSchemaDiff(old, updated))
	})
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "single type", raw: `"string"`, want: "string"},
		{name: "nullable array", raw: `["null","string"]`, want: "string"},
		{name: "union is sorted", raw: `["string","integer","null"]`, want: "integer,string"},
		{name: "unknown shape falls back to raw", raw: `{"x":1}`, want: `{"x":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeType(json.RawMessage(tt.raw)))
		})
	}
}

func TestParseColumns(t *testing.T) {
	assert.Nil(t, parseColumns(nil))
	assert.Nil(t, parseColumns(json.RawMessage(`not json`)))
	assert.Equal(t,
		map[string]string{"id": "integer", "broken": "", "untyped": ""},
		parseColumns(json.RawMessage(`{"properties":{"id":{"type":"integer"},"broken":"x","untyped":{}}}`)),
	)
}
