package appmcp

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxSchemaDepth = 6

// messageSchema is the JSON Schema of the protojson form of a message, with
// fields under their JSON names. Nesting deeper than maxSchemaDepth is an
// unconstrained object.
func messageSchema(message protoreflect.MessageDescriptor) map[string]any {
	return objectSchema(message, 0)
}

func objectSchema(message protoreflect.MessageDescriptor, depth int) map[string]any {
	if depth >= maxSchemaDepth {
		return map[string]any{"type": "object"}
	}

	properties := map[string]any{}

	fields := message.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		properties[field.JSONName()] = fieldSchema(field, depth)
	}

	return map[string]any{"type": "object", "properties": properties}
}

func fieldSchema(field protoreflect.FieldDescriptor, depth int) map[string]any {
	switch {
	case field.IsMap():
		return map[string]any{
			"type":                 "object",
			"additionalProperties": singularSchema(field.MapValue(), depth),
		}
	case field.IsList():
		return map[string]any{"type": "array", "items": singularSchema(field, depth)}
	default:
		return singularSchema(field, depth)
	}
}

func singularSchema(field protoreflect.FieldDescriptor, depth int) map[string]any {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return map[string]any{"type": "boolean"}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return map[string]any{"type": "integer"}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return map[string]any{"type": []string{"integer", "string"}}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return map[string]any{"type": "number"}
	case protoreflect.StringKind:
		return map[string]any{"type": "string"}
	case protoreflect.BytesKind:
		return map[string]any{"type": "string", "contentEncoding": "base64"}
	case protoreflect.EnumKind:
		return enumSchema(field.Enum())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageFieldSchema(field.Message(), depth+1)
	default:
		return map[string]any{}
	}
}

func enumSchema(enum protoreflect.EnumDescriptor) map[string]any {
	values := enum.Values()
	names := make([]string, 0, values.Len())

	for i := range values.Len() {
		names = append(names, string(values.Get(i).Name()))
	}

	return map[string]any{"type": "string", "enum": names}
}

func messageFieldSchema(message protoreflect.MessageDescriptor, depth int) map[string]any {
	switch message.FullName() {
	case "google.protobuf.Timestamp":
		return map[string]any{"type": "string", "format": "date-time"}
	case "google.protobuf.Duration":
		return map[string]any{"type": "string", "description": `seconds with an "s" suffix, e.g. "3.5s"`}
	case "google.protobuf.FieldMask":
		return map[string]any{"type": "string"}
	case "google.protobuf.Struct":
		return map[string]any{"type": "object"}
	case "google.protobuf.ListValue":
		return map[string]any{"type": "array"}
	case "google.protobuf.Value", "google.protobuf.Any":
		return map[string]any{}
	case "google.protobuf.Empty":
		return map[string]any{"type": "object"}
	case "google.protobuf.StringValue":
		return map[string]any{"type": "string"}
	case "google.protobuf.BoolValue":
		return map[string]any{"type": "boolean"}
	case "google.protobuf.Int32Value", "google.protobuf.UInt32Value":
		return map[string]any{"type": "integer"}
	case "google.protobuf.Int64Value", "google.protobuf.UInt64Value":
		return map[string]any{"type": []string{"integer", "string"}}
	case "google.protobuf.FloatValue", "google.protobuf.DoubleValue":
		return map[string]any{"type": "number"}
	case "google.protobuf.BytesValue":
		return map[string]any{"type": "string", "contentEncoding": "base64"}
	default:
		return objectSchema(message, depth)
	}
}
