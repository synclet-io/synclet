package pipelinecatalog

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// SchemaChangeType identifies the kind of schema change detected.
type SchemaChangeType string

const (
	StreamAdded       SchemaChangeType = "stream_added"
	StreamRemoved     SchemaChangeType = "stream_removed"
	ColumnAdded       SchemaChangeType = "column_added"
	ColumnRemoved     SchemaChangeType = "column_removed"
	ColumnTypeChanged SchemaChangeType = "column_type_changed"
)

// SchemaChange describes a single difference between two catalogs.
type SchemaChange struct {
	Type       SchemaChangeType `json:"type"`
	StreamName string           `json:"stream_name"`
	Namespace  string           `json:"namespace,omitempty"`
	ColumnName string           `json:"column_name,omitempty"`
	OldType    string           `json:"old_type,omitempty"`
	NewType    string           `json:"new_type,omitempty"`
}

// jsonSchemaObject is a minimal representation of a JSON Schema object with properties.
type jsonSchemaObject struct {
	Properties map[string]json.RawMessage `json:"properties"`
}

// normalizeType converts a JSON schema type value into a canonical string for comparison.
// It handles both single types ("string") and arrays (["string", "null"]).
// Nullable markers are stripped so that ["string", "null"] normalizes to "string".
func normalizeType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try unmarshalling as a string first.
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return single
	}

	// Try unmarshalling as an array of strings.
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		// Filter out "null" -- it indicates nullability, not the data type.
		var types []string
		for _, t := range arr {
			if t != "null" {
				types = append(types, t)
			}
		}
		sort.Strings(types)
		return strings.Join(types, ",")
	}

	// Fallback: return the raw JSON string for comparison.
	return string(raw)
}

// columnTypeDescriptor is a small helper struct to hold the raw type field from a property schema.
type columnTypeDescriptor struct {
	Type json.RawMessage `json:"type"`
}

// parseColumns extracts column names and their normalized type strings from a JSON schema.
func parseColumns(schema json.RawMessage) map[string]string {
	if len(schema) == 0 {
		return nil
	}

	var obj jsonSchemaObject
	if err := json.Unmarshal(schema, &obj); err != nil {
		return nil
	}

	columns := make(map[string]string, len(obj.Properties))
	for name, propRaw := range obj.Properties {
		var desc columnTypeDescriptor
		if err := json.Unmarshal(propRaw, &desc); err != nil {
			columns[name] = ""
			continue
		}
		columns[name] = normalizeType(desc.Type)
	}
	return columns
}

// ComputeSchemaDiff compares an old catalog with a new catalog and returns a list of changes.
// Both arguments may be nil; a nil catalog is treated as having zero streams.
func ComputeSchemaDiff(old, updated *protocol.AirbyteCatalog) []SchemaChange {
	oldStreams := buildStreamMap(old)
	newStreams := buildStreamMap(updated)

	var changes []SchemaChange

	// Detect removed and modified streams.
	for key, oldStream := range oldStreams {
		newStream, exists := newStreams[key]
		if !exists {
			changes = append(changes, SchemaChange{
				Type:       StreamRemoved,
				StreamName: oldStream.Name,
				Namespace:  oldStream.Namespace,
			})
			continue
		}

		// Stream exists in both -- compare columns.
		changes = append(changes, diffColumns(oldStream, newStream)...)
	}

	// Detect added streams.
	for key, newStream := range newStreams {
		if _, exists := oldStreams[key]; !exists {
			changes = append(changes, SchemaChange{
				Type:       StreamAdded,
				StreamName: newStream.Name,
				Namespace:  newStream.Namespace,
			})
		}
	}

	// Sort for deterministic output.
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].StreamName != changes[j].StreamName {
			return changes[i].StreamName < changes[j].StreamName
		}
		if changes[i].Namespace != changes[j].Namespace {
			return changes[i].Namespace < changes[j].Namespace
		}
		if changes[i].Type != changes[j].Type {
			return changes[i].Type < changes[j].Type
		}
		return changes[i].ColumnName < changes[j].ColumnName
	})

	return changes
}

// buildStreamMap indexes catalog streams by their unique key.
func buildStreamMap(catalog *protocol.AirbyteCatalog) map[string]protocol.AirbyteStream {
	if catalog == nil {
		return nil
	}
	m := make(map[string]protocol.AirbyteStream, len(catalog.Streams))
	for _, s := range catalog.Streams {
		m[streamKey(s.Namespace, s.Name)] = s
	}
	return m
}

// diffColumns compares the JSON schema properties of two streams and returns column-level changes.
func diffColumns(oldStream, newStream protocol.AirbyteStream) []SchemaChange {
	oldCols := parseColumns(oldStream.JSONSchema)
	newCols := parseColumns(newStream.JSONSchema)

	var changes []SchemaChange

	// Detect removed and changed columns.
	for col, oldType := range oldCols {
		newType, exists := newCols[col]
		if !exists {
			changes = append(changes, SchemaChange{
				Type:       ColumnRemoved,
				StreamName: oldStream.Name,
				Namespace:  oldStream.Namespace,
				ColumnName: col,
			})
			continue
		}
		if oldType != newType {
			changes = append(changes, SchemaChange{
				Type:       ColumnTypeChanged,
				StreamName: oldStream.Name,
				Namespace:  oldStream.Namespace,
				ColumnName: col,
				OldType:    oldType,
				NewType:    newType,
			})
		}
	}

	// Detect added columns.
	for col := range newCols {
		if _, exists := oldCols[col]; !exists {
			changes = append(changes, SchemaChange{
				Type:       ColumnAdded,
				StreamName: oldStream.Name,
				Namespace:  oldStream.Namespace,
				ColumnName: col,
			})
		}
	}

	return changes
}
