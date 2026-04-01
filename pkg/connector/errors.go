package connector

import (
	"fmt"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// connectorError represents an error reported by a connector via a TRACE error message.
type connectorError struct {
	// Message is the user-facing error message from the connector.
	Message string
	// FailureType indicates whether the error is a config error or a system error.
	FailureType protocol.FailureType
}

func (e *connectorError) Error() string {
	if e.FailureType != "" {
		return fmt.Sprintf("connector error (%s): %s", e.FailureType, e.Message)
	}

	return "connector error: " + e.Message
}

// ExitCodeError wraps a connector exit with a typed exit code for classification.
// Enables downstream code to extract exit codes via errors.As for retry decisions.
type ExitCodeError struct {
	ExitCode int
	Role     string // "source" or "destination"
	Stderr   string
}

func (e *ExitCodeError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s connector exited with code %d: %s", e.Role, e.ExitCode, e.Stderr)
	}

	return fmt.Sprintf("%s connector exited with code %d", e.Role, e.ExitCode)
}
