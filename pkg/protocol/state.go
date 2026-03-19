package protocol

import (
	"encoding/json"
	"strings"
)

// AirbyteStateType represents the type of state message.
type AirbyteStateType string

const (
	StateTypeStream AirbyteStateType = "stream"
	StateTypeGlobal AirbyteStateType = "global"
	StateTypeLegacy AirbyteStateType = "legacy"
)

// NormalizeStateType converts an Airbyte state type to the canonical lowercase form
// used in the database. Airbyte connectors emit UPPERCASE ("STREAM", "GLOBAL", "LEGACY")
// but our Postgres enum uses lowercase.
func NormalizeStateType(t AirbyteStateType) AirbyteStateType {
	switch strings.ToLower(string(t)) {
	case "stream":
		return StateTypeStream
	case "global":
		return StateTypeGlobal
	case "legacy":
		return StateTypeLegacy
	default:
		return AirbyteStateType(strings.ToLower(string(t)))
	}
}

// AirbyteStateMessage represents a state checkpoint message.
type AirbyteStateMessage struct {
	Type             AirbyteStateType    `json:"type"`
	Stream           *AirbyteStreamState `json:"stream,omitempty"`
	Global           *AirbyteGlobalState `json:"global,omitempty"`
	Data             json.RawMessage     `json:"data,omitempty"`
	SourceStats      *StateStats         `json:"sourceStats,omitempty"`
	DestinationStats *StateStats         `json:"destinationStats,omitempty"`
}

// StateStats contains record count statistics per state checkpoint.
type StateStats struct {
	RecordCount float64 `json:"recordCount"`
}

// AirbyteStreamState represents per-stream state.
type AirbyteStreamState struct {
	StreamDescriptor StreamDescriptor `json:"stream_descriptor"`
	StreamState      json.RawMessage  `json:"stream_state,omitempty"`
}

// AirbyteGlobalState represents global state shared across streams.
type AirbyteGlobalState struct {
	SharedState  json.RawMessage      `json:"shared_state,omitempty"`
	StreamStates []AirbyteStreamState `json:"stream_states"`
}

// StreamDescriptor identifies a stream by name and optional namespace.
type StreamDescriptor struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}
