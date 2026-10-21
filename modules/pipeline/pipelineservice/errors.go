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
	ErrWorkflowRunActive      baseErr = "the workflow has an active run; wait for it to end"
	ErrWorkflowRunNotFailed   baseErr = "only a failed workflow run can be rerun"
	ErrConnectionInWorkflow   baseErr = "the connection is a step of a workflow; remove it from the workflow first"
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

// Roles of the connector an ExitCodeError reports.
const (
	ExitRoleSource      = "source"
	ExitRoleDestination = "destination"
)

// ExitCodeError wraps a connector exit with a typed exit code for classification.
// Enables downstream code to extract exit codes via errors.As for retry decisions.
type ExitCodeError struct {
	ExitCode int
	Role     string // ExitRoleSource or ExitRoleDestination
	Stderr   string
}

func (e *ExitCodeError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s connector exited with code %d: %s", e.Role, e.ExitCode, e.Stderr)
	}

	return fmt.Sprintf("%s connector exited with code %d", e.Role, e.ExitCode)
}

// ExitCodeErrors returns every ExitCodeError in err's tree, depth first.
func ExitCodeErrors(err error) []*ExitCodeError {
	switch typed := err.(type) { //nolint:errorlint // walks the whole tree, errors.As stops at the first match
	case nil:
		return nil
	case *ExitCodeError:
		return []*ExitCodeError{typed}
	case interface{ Unwrap() error }:
		return ExitCodeErrors(typed.Unwrap())
	case interface{ Unwrap() []error }:
		var found []*ExitCodeError
		for _, wrapped := range typed.Unwrap() {
			found = append(found, ExitCodeErrors(wrapped)...)
		}

		return found
	default:
		return nil
	}
}

// ReportedFailureError is a sync failure an executor reported by message and
// connector exit codes. Its text is the message; it unwraps to an
// ExitCodeError per connector that exited with a non-zero code.
type ReportedFailureError struct {
	Message string
	Exits   []error
}

// NewReportedFailureError builds a ReportedFailureError; a zero exit code
// means the connector exited cleanly or reported none.
func NewReportedFailureError(message string, sourceExitCode, destExitCode int) *ReportedFailureError {
	failure := &ReportedFailureError{Message: message}

	if sourceExitCode != 0 {
		failure.Exits = append(failure.Exits, &ExitCodeError{ExitCode: sourceExitCode, Role: ExitRoleSource})
	}

	if destExitCode != 0 {
		failure.Exits = append(failure.Exits, &ExitCodeError{ExitCode: destExitCode, Role: ExitRoleDestination})
	}

	return failure
}

func (e *ReportedFailureError) Error() string { return e.Message }

func (e *ReportedFailureError) Unwrap() []error { return e.Exits }
