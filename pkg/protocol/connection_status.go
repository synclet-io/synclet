package protocol

// ConnectionStatusType represents the result of a connection check.
type ConnectionStatusType string

const (
	ConnectionStatusSucceeded ConnectionStatusType = "SUCCEEDED"
	ConnectionStatusFailed    ConnectionStatusType = "FAILED"
)

// AirbyteConnectionStatus represents the result of a check connection operation.
type AirbyteConnectionStatus struct {
	Status  ConnectionStatusType `json:"status"`
	Message string               `json:"message,omitempty"`
}
