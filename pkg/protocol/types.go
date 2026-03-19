package protocol

import "encoding/json"

// MessageType represents the type discriminator for AirbyteMessage.
type MessageType string

const (
	MessageTypeRecord              MessageType = "RECORD"
	MessageTypeState               MessageType = "STATE"
	MessageTypeLog                 MessageType = "LOG"
	MessageTypeSpec                MessageType = "SPEC"
	MessageTypeCatalog             MessageType = "CATALOG"
	MessageTypeConnectionStatus    MessageType = "CONNECTION_STATUS"
	MessageTypeTrace               MessageType = "TRACE"
	MessageTypeControl             MessageType = "CONTROL"
	MessageTypeDestinationCatalog  MessageType = "DESTINATION_CATALOG"
)

// AirbyteMessage is the top-level envelope for all Airbyte protocol messages.
type AirbyteMessage struct {
	Type                MessageType              `json:"type"`
	Record              *AirbyteRecordMessage    `json:"record,omitempty"`
	State               *AirbyteStateMessage     `json:"state,omitempty"`
	Log                 *AirbyteLogMessage       `json:"log,omitempty"`
	Spec                *ConnectorSpecification  `json:"spec,omitempty"`
	Catalog             *AirbyteCatalog          `json:"catalog,omitempty"`
	ConnectionStatus    *AirbyteConnectionStatus `json:"connectionStatus,omitempty"`
	Trace               *AirbyteTraceMessage     `json:"trace,omitempty"`
	Control             *AirbyteControlMessage   `json:"control,omitempty"`
	DestinationCatalog  *DestinationCatalog      `json:"destinationCatalog,omitempty"`
}

// DestinationCatalog describes the catalog reported by a destination for file-based operations.
type DestinationCatalog struct {
	Catalog *AirbyteCatalog `json:"catalog"`
}

// ControlMessageType represents the type of control message.
type ControlMessageType string

const (
	ControlMessageTypeConnectorConfig ControlMessageType = "CONNECTOR_CONFIG"
)

// AirbyteControlMessage represents a control message from a connector.
type AirbyteControlMessage struct {
	Type            ControlMessageType              `json:"type"`
	EmittedAt       float64                         `json:"emitted_at"`
	ConnectorConfig *AirbyteControlConnectorConfig  `json:"connectorConfig,omitempty"`
}

// AirbyteControlConnectorConfig contains an updated connector configuration.
type AirbyteControlConnectorConfig struct {
	Config json.RawMessage `json:"config"`
}
