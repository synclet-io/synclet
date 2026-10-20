package mapper

import (
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/google/cel-go/common/types"
)

const (
	recordTypeName = "mapper.r"
	syncTypeName   = "mapper.sync"
)

var (
	schemaString    = json.RawMessage(`{"type":"string"}`)
	schemaInteger   = json.RawMessage(`{"type":"integer"}`)
	schemaNumber    = json.RawMessage(`{"type":"number"}`)
	schemaBoolean   = json.RawMessage(`{"type":"boolean"}`)
	schemaTimestamp = json.RawMessage(`{"type":"string","format":"date-time","airbyte_type":"timestamp_with_timezone"}`)
	schemaDate      = json.RawMessage(`{"type":"string","format":"date"}`)
	schemaObject    = json.RawMessage(`{"type":"object"}`)
	schemaArray     = json.RawMessage(`{"type":"array"}`)
	schemaAny       = json.RawMessage(`{}`)
)

// schemaNode is the part of a JSON schema the mapper reads.
type schemaNode struct {
	Type        any                        `json:"type"`
	Format      string                     `json:"format"`
	AirbyteType string                     `json:"airbyte_type"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Items       json.RawMessage            `json:"items"`
}

func parseNode(raw json.RawMessage) schemaNode {
	var node schemaNode
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &node) //nolint:errcheck // an unreadable node is treated as any type
	}

	return node
}

// singleType returns the node's one non-null type, or "" when it has none or several.
func (n schemaNode) singleType() string {
	switch typ := n.Type.(type) {
	case string:
		return typ
	case []any:
		found := ""

		for _, item := range typ {
			name, ok := item.(string)
			if !ok || name == "null" {
				continue
			}

			if found != "" {
				return ""
			}

			found = name
		}

		return found
	}

	return ""
}

// recordTypes is the CEL type provider of one step: the record's object types
// built from the step's input schema, on top of the standard registry.
type recordTypes struct {
	types.Provider

	structs map[string]map[string]*types.Type
	sources map[string]json.RawMessage
}

func newRecordTypes(props map[string]json.RawMessage) (*recordTypes, *types.Type, error) {
	base, err := types.NewRegistry()
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // registry construction has no context to add
	}

	provider := &recordTypes{
		Provider: base,
		structs:  map[string]map[string]*types.Type{},
		sources:  map[string]json.RawMessage{},
	}

	provider.structs[syncTypeName] = map[string]*types.Type{
		"started_at": types.TimestampType,
		"job_id":     types.StringType,
		"connection": types.StringType,
	}

	if props == nil {
		return provider, types.DynType, nil
	}

	provider.register(recordTypeName, props)

	return provider, types.NewObjectType(recordTypeName), nil
}

func (p *recordTypes) register(name string, props map[string]json.RawMessage) {
	fields := make(map[string]*types.Type, len(props))
	p.structs[name] = fields

	for field, raw := range props {
		fields[field] = p.typeOf(name+"."+field, raw)
	}
}

func (p *recordTypes) typeOf(name string, raw json.RawMessage) *types.Type {
	node := parseNode(raw)

	switch node.singleType() {
	case "integer":
		return types.IntType
	case "number":
		return types.DoubleType
	case "boolean":
		return types.BoolType
	case "string":
		return types.StringType
	case "array":
		if len(node.Items) == 0 {
			return types.NewListType(types.DynType)
		}

		return types.NewListType(p.typeOf(name+"[]", node.Items))
	case "object", "":
		if len(node.Properties) == 0 {
			return types.DynType
		}

		p.register(name, node.Properties)
		p.sources[name] = raw

		return types.NewObjectType(name)
	}

	return types.DynType
}

func (p *recordTypes) FindStructType(structType string) (*types.Type, bool) {
	if _, ok := p.structs[structType]; ok {
		return types.NewTypeTypeWithParam(types.NewObjectType(structType)), true
	}

	return p.Provider.FindStructType(structType)
}

func (p *recordTypes) FindStructFieldNames(structType string) ([]string, bool) {
	fields, ok := p.structs[structType]
	if !ok {
		return p.Provider.FindStructFieldNames(structType)
	}

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}

	return names, true
}

func (p *recordTypes) FindStructFieldType(structType, fieldName string) (*types.FieldType, bool) {
	fields, ok := p.structs[structType]
	if !ok {
		return p.Provider.FindStructFieldType(structType, fieldName)
	}

	typ, ok := fields[fieldName]
	if !ok {
		return nil, false
	}

	return &types.FieldType{Type: typ}, true
}

// schemaOf returns the JSON schema of a value of the CEL type.
func (p *recordTypes) schemaOf(typ *types.Type) json.RawMessage {
	switch typ.Kind() {
	case types.IntKind, types.UintKind:
		return schemaInteger
	case types.DoubleKind:
		return schemaNumber
	case types.BoolKind:
		return schemaBoolean
	case types.StringKind, types.BytesKind, types.DurationKind:
		return schemaString
	case types.TimestampKind:
		return schemaTimestamp
	case types.ListKind:
		elem := typ.Parameters()[0]
		if elem.Kind() == types.DynKind {
			return schemaArray
		}

		out, _ := json.Marshal(map[string]json.RawMessage{ //nolint:errcheck // raw messages always marshal
			"type":  json.RawMessage(`"array"`),
			"items": p.schemaOf(elem),
		})

		return out
	case types.StructKind:
		if raw, ok := p.sources[typ.TypeName()]; ok {
			return raw
		}

		return schemaObject
	default:
		return schemaAny
	}
}

// stringSchemaLike is the schema of a string written in place of a field,
// nullable when the field was or when the step can write null.
func stringSchemaLike(raw json.RawMessage, canBeNull bool) json.RawMessage {
	return withNull(schemaString, canBeNull || parseNode(raw).nullable())
}

// withNull adds "null" to a schema's type when canBeNull; a schema without a
// type already admits null.
func withNull(raw json.RawMessage, canBeNull bool) json.RawMessage {
	if !canBeNull {
		return raw
	}

	node := map[string]any{}
	if err := json.Unmarshal(raw, &node); err != nil {
		return raw
	}

	switch typ := node["type"].(type) {
	case string:
		if typ == "null" {
			return raw
		}

		node["type"] = []any{"null", typ}
	case []any:
		if slices.Contains(typ, any("null")) {
			return raw
		}

		node["type"] = append([]any{"null"}, typ...)
	default:
		return raw
	}

	return mustMarshal(node)
}

func (n schemaNode) nullable() bool {
	list, ok := n.Type.([]any)

	return ok && slices.Contains(list, any("null"))
}

func castSchema(target string) json.RawMessage {
	switch target {
	case CastString:
		return schemaString
	case CastInteger:
		return schemaInteger
	case CastNumber:
		return schemaNumber
	case CastBoolean:
		return schemaBoolean
	case CastTimestamp:
		return schemaTimestamp
	case CastDate:
		return schemaDate
	case CastObject:
		return schemaObject
	case CastArray:
		return schemaArray
	}

	return nil
}

// shape tells how to decode the numbers of a record per its schema.
type shape struct {
	kind   string
	fields map[string]*shape
	items  *shape
}

func shapeOf(props map[string]json.RawMessage) *shape {
	out := &shape{kind: "object", fields: make(map[string]*shape, len(props))}
	for field, raw := range props {
		out.fields[field] = shapeOfNode(parseNode(raw))
	}

	return out
}

func shapeOfNode(node schemaNode) *shape {
	switch typ := node.singleType(); typ {
	case "integer", "number":
		return &shape{kind: typ}
	case "array":
		if len(node.Items) == 0 {
			return nil
		}

		return &shape{kind: typ, items: shapeOfNode(parseNode(node.Items))}
	default:
		if len(node.Properties) == 0 {
			return nil
		}

		return shapeOf(node.Properties)
	}
}

// normalize copies a decoded value, turning its numbers into int64 or float64
// as the shape says, and into int64 when integral where the shape says nothing.
func normalize(value any, shp *shape) any {
	kind := ""
	if shp != nil {
		kind = shp.kind
	}

	switch val := value.(type) {
	case json.Number:
		if kind != "number" {
			if n, err := strconv.ParseInt(string(val), 10, 64); err == nil {
				return n
			}
		}

		f, err := val.Float64()
		if err != nil {
			return string(val)
		}

		return normalize(f, shp)
	case float64:
		if kind == "integer" && val == math.Trunc(val) && math.Abs(val) < 1<<63 {
			return int64(val)
		}

		return val
	case float32:
		return normalize(float64(val), shp)
	case int:
		return numberOf(int64(val), kind)
	case int32:
		return numberOf(int64(val), kind)
	case int64:
		return numberOf(val, kind)
	case map[string]any:
		out := make(map[string]any, len(val))

		for key, item := range val {
			var sub *shape
			if shp != nil {
				sub = shp.fields[key]
			}

			out[key] = normalize(item, sub)
		}

		return out
	case []any:
		out := make([]any, len(val))

		for i, item := range val {
			var sub *shape
			if shp != nil {
				sub = shp.items
			}

			out[i] = normalize(item, sub)
		}

		return out
	default:
		return value
	}
}

func numberOf(val int64, kind string) any {
	if kind == "number" {
		return float64(val)
	}

	return val
}

// dateKind names the date, time or date-time string a node holds by its format
// or Airbyte type, or returns "" for any other node.
func (n schemaNode) dateKind() string {
	if n.singleType() != "string" {
		return ""
	}

	switch {
	case n.Format == "date-time", strings.HasPrefix(n.AirbyteType, "timestamp"):
		return "date-time"
	case n.Format == "date", n.AirbyteType == "date":
		return "date"
	case n.Format == "time", strings.HasPrefix(n.AirbyteType, "time_"):
		return "time"
	}

	return ""
}
