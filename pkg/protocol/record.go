package protocol

import "encoding/json"

// AirbyteRecordMessage represents a data record from a source stream.
type AirbyteRecordMessage struct {
	Stream        string         `json:"stream"`
	Data          json.RawMessage `json:"data"`
	EmittedAt     int64           `json:"emitted_at"`
	Namespace     string          `json:"namespace,omitempty"`
	Meta          *RecordMeta     `json:"meta,omitempty"`
	FileReference *FileReference  `json:"file_reference,omitempty"`
}

// FileReference contains staging information for file-based transfers.
type FileReference struct {
	StagingFilePath string `json:"staging_file_path"`
	SourceFileURL   string `json:"source_file_url,omitempty"`
	FileSize        int64  `json:"file_size,omitempty"`
}

// Change represents a change type for schema evolution tracking.
type Change string

const (
	ChangeInsert Change = "INSERT"
	ChangeUpdate Change = "UPDATE"
	ChangeDelete Change = "DELETE"
)

// FieldChange tracks a change to a specific field in a record.
type FieldChange struct {
	Field  string `json:"field"`
	Change Change `json:"change"`
	Reason string `json:"reason,omitempty"`
}

// RecordMeta contains metadata about a record, including schema evolution changes.
type RecordMeta struct {
	Changes []FieldChange `json:"changes,omitempty"`
}
