package pipelinecatalog

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// ValidateSelectedFields checks that all field paths in selectedFields exist
// in the provided JSON schema. Returns an error for the first invalid path found.
func ValidateSelectedFields(selectedFields []protocol.SelectedField, jsonSchema json.RawMessage) error {
	var schema schemaNode
	if err := json.Unmarshal(jsonSchema, &schema); err != nil {
		return fmt.Errorf("parsing json schema: %w", err)
	}

	for _, selectedField := range selectedFields {
		if len(selectedField.FieldPath) == 0 {
			return fmt.Errorf("empty field path in selected fields")
		}

		if !fieldPathExists(schema, selectedField.FieldPath) {
			return fmt.Errorf("field path %v not found in schema", selectedField.FieldPath)
		}
	}

	return nil
}

// ForceIncludeFields ensures that cursor and primary key field paths are
// included in the selected fields list. Returns the updated list.
func ForceIncludeFields(
	selectedFields []protocol.SelectedField,
	cursorField []string,
	primaryKey [][]string,
) []protocol.SelectedField {
	result := slices.Clone(selectedFields)

	// Force-include cursor field if present.
	if len(cursorField) > 0 && !containsFieldPath(result, cursorField) {
		result = append(result, protocol.SelectedField{FieldPath: cursorField})
	}

	// Force-include each primary key column path.
	for _, pkPath := range primaryKey {
		if len(pkPath) > 0 && !containsFieldPath(result, pkPath) {
			result = append(result, protocol.SelectedField{FieldPath: pkPath})
		}
	}

	return result
}

// filterSchemaBySelectedFields returns a new JSON schema containing only
// the properties matching the selected field paths.
func filterSchemaBySelectedFields(jsonSchema json.RawMessage, selectedFields []protocol.SelectedField) (json.RawMessage, error) {
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(jsonSchema, &schema); err != nil {
		return nil, fmt.Errorf("parsing json schema: %w", err)
	}

	propsRaw, ok := schema["properties"]
	if !ok {
		return jsonSchema, nil
	}

	var props map[string]json.RawMessage
	if err := json.Unmarshal(propsRaw, &props); err != nil {
		return nil, fmt.Errorf("parsing properties: %w", err)
	}

	// Build set of top-level field names to keep.
	keep := make(map[string]bool)

	for _, sf := range selectedFields {
		if len(sf.FieldPath) > 0 {
			keep[sf.FieldPath[0]] = true
		}
	}

	filtered := make(map[string]json.RawMessage)

	for name, prop := range props {
		if keep[name] {
			filtered[name] = prop
		}
	}

	filteredPropsJSON, err := json.Marshal(filtered)
	if err != nil {
		return nil, fmt.Errorf("marshaling filtered properties: %w", err)
	}

	schema["properties"] = filteredPropsJSON

	result, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshaling filtered schema: %w", err)
	}

	return result, nil
}

// FilterRecordData filters a record's data JSON to only include the selected
// field paths (top-level fields).
func FilterRecordData(data json.RawMessage, selectedFields []protocol.SelectedField) (json.RawMessage, error) {
	var record map[string]json.RawMessage
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parsing record data: %w", err)
	}

	keep := make(map[string]bool)

	for _, sf := range selectedFields {
		if len(sf.FieldPath) > 0 {
			keep[sf.FieldPath[0]] = true
		}
	}

	filtered := make(map[string]json.RawMessage)

	for name, val := range record {
		if keep[name] {
			filtered[name] = val
		}
	}

	result, err := json.Marshal(filtered)
	if err != nil {
		return nil, fmt.Errorf("marshaling filtered record: %w", err)
	}

	return result, nil
}

// BuildDestinationCatalog creates a deep copy of the catalog with JSON schemas
// filtered by selected fields for streams that have them. Streams without
// selected fields are passed through unchanged.
func BuildDestinationCatalog(catalog *protocol.ConfiguredAirbyteCatalog) (*protocol.ConfiguredAirbyteCatalog, error) {
	dest := &protocol.ConfiguredAirbyteCatalog{
		Streams: make([]protocol.ConfiguredAirbyteStream, len(catalog.Streams)),
	}

	for i, stream := range catalog.Streams {
		dest.Streams[i] = stream

		if len(stream.SelectedFields) == 0 {
			continue
		}

		filteredSchema, err := filterSchemaBySelectedFields(stream.Stream.JSONSchema, stream.SelectedFields)
		if err != nil {
			return nil, fmt.Errorf("filtering schema for stream %q: %w", stream.Stream.Name, err)
		}

		dest.Streams[i].Stream.JSONSchema = filteredSchema
	}

	return dest, nil
}

// schemaNode is a minimal representation of a JSON schema for field path validation.
type schemaNode struct {
	Properties map[string]schemaNode `json:"properties,omitempty"`
}

// fieldPathExists checks whether a field path exists in the schema.
func fieldPathExists(node schemaNode, path []string) bool {
	if len(path) == 0 {
		return true
	}

	child, ok := node.Properties[path[0]]
	if !ok {
		return false
	}

	return fieldPathExists(child, path[1:])
}

// containsFieldPath checks if the selected fields list already contains a path.
func containsFieldPath(fields []protocol.SelectedField, path []string) bool {
	for _, f := range fields {
		if slices.Equal(f.FieldPath, path) {
			return true
		}
	}

	return false
}
