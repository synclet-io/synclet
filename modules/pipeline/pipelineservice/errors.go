package pipelineservice

import "fmt"

type baseErr string

func (e baseErr) Error() string { return string(e) }

const (
	ErrStateDataInvalidJSON   baseErr = "state_data must be valid JSON"
	ErrConnectorNotLinked     baseErr = "connector is not linked to a repository"
	ErrMissingCheckTaskParams baseErr = "either source_id/destination_id or managed_connector_id+config must be provided"
	ErrEmptyFieldPath         baseErr = "empty field path in selected fields"
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
