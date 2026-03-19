package protocol

import "encoding/json"

// SyncMode represents how a stream should be synced.
type SyncMode string

const (
	SyncModeFullRefresh SyncMode = "full_refresh"
	SyncModeIncremental SyncMode = "incremental"
)

// DestinationSyncMode represents how records are written to the destination.
type DestinationSyncMode string

const (
	DestinationSyncModeOverwrite   DestinationSyncMode = "overwrite"
	DestinationSyncModeAppend      DestinationSyncMode = "append"
	DestinationSyncModeAppendDedup DestinationSyncMode = "append_dedup"
	DestinationSyncModeUpdate      DestinationSyncMode = "update"
	DestinationSyncModeSoftDelete  DestinationSyncMode = "soft_delete"
)

// SelectedField represents a field path selected for replication.
type SelectedField struct {
	FieldPath []string `json:"field_path"`
}

// AirbyteCatalog represents the full catalog of streams available from a source.
type AirbyteCatalog struct {
	Streams []AirbyteStream `json:"streams"`
}

// AirbyteStream describes a single stream in the catalog.
type AirbyteStream struct {
	Name                    string          `json:"name"`
	JSONSchema              json.RawMessage `json:"json_schema"`
	SupportedSyncModes      []SyncMode      `json:"supported_sync_modes,omitempty"`
	SourceDefinedCursor     bool            `json:"source_defined_cursor,omitempty"`
	DefaultCursorField      []string        `json:"default_cursor_field,omitempty"`
	SourceDefinedPrimaryKey [][]string      `json:"source_defined_primary_key,omitempty"`
	Namespace               string          `json:"namespace,omitempty"`
	IsResumable             bool            `json:"is_resumable,omitempty"`
	IsFileBased             bool            `json:"is_file_based,omitempty"`
}

// ConfiguredAirbyteCatalog is the user-configured catalog sent to a source for syncing.
type ConfiguredAirbyteCatalog struct {
	Streams []ConfiguredAirbyteStream `json:"streams"`
}

// ConfiguredAirbyteStream is a user-configured stream with sync settings.
type ConfiguredAirbyteStream struct {
	Stream                AirbyteStream       `json:"stream"`
	SyncMode              SyncMode            `json:"sync_mode"`
	DestinationSyncMode   DestinationSyncMode `json:"destination_sync_mode"`
	CursorField           []string            `json:"cursor_field,omitempty"`
	PrimaryKey            [][]string          `json:"primary_key,omitempty"`
	DestinationObjectName string              `json:"destination_object_name,omitempty"`
	GenerationID          int64               `json:"generation_id"`
	MinimumGenerationID   int64               `json:"minimum_generation_id"`
	SyncID                int64               `json:"sync_id"`
	IncludeFiles          bool                `json:"include_files,omitempty"`
	SelectedFields        []SelectedField     `json:"selected_fields,omitempty"`
}
