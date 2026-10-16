package pipelineservice

import (
	"fmt"
	"strings"
)

type baseErr string

func (e baseErr) Error() string { return string(e) }

const (
	ErrStateDataInvalidJSON   baseErr = "state_data must be valid JSON"
	ErrConnectorNotLinked     baseErr = "connector is not linked to a repository"
	ErrMissingCheckTaskParams baseErr = "either source_id/destination_id or managed_connector_id+config must be provided"
	ErrEmptyFieldPath         baseErr = "empty field path in selected fields"
	ErrNoCachedSpec           baseErr = "connector has no cached spec yet; wait for the repository sync to finish"
)

// ValidationError indicates invalid input.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
	}

	return "validation error: " + e.Message
}

// InUseError reports a resource that connections still reference and so
// cannot be deleted.
type InUseError struct {
	Resource        string
	ConnectionNames []string
}

func (e *InUseError) Error() string {
	return fmt.Sprintf("%s is used by connections: %s; delete those connections first",
		e.Resource, strings.Join(e.ConnectionNames, ", "))
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
