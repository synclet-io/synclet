package pipelinejobs

import "fmt"

// FailureCategory classifies container exit codes into categories that
// determine retry behavior for failed sync jobs.
type FailureCategory string

const (
	// FailurePermanent indicates a non-recoverable failure (e.g., OOMKilled, SIGSEGV).
	// Jobs with permanent failures should not be retried.
	FailurePermanent FailureCategory = "permanent"

	// FailureTransient indicates a potentially recoverable failure.
	// Jobs with transient failures can be retried with existing backoff logic.
	FailureTransient FailureCategory = "transient"

	// FailureIntentional indicates the container was intentionally killed (e.g., SIGTERM).
	// Jobs with intentional failures should not be retried.
	FailureIntentional FailureCategory = "intentional"
)

// ClassifyExitCode maps a container exit code to a failure category and
// a human-readable reason string suitable for storing in the job's FailureReason field.
func ClassifyExitCode(exitCode int) (category FailureCategory, reason string) {
	switch exitCode {
	case 137:
		return FailurePermanent, "OOMKilled (exit 137)"
	case 139:
		return FailurePermanent, "SIGSEGV (exit 139)"
	case 143:
		return FailureIntentional, "SIGTERM (exit 143)"
	default:
		return FailureTransient, fmt.Sprintf("exit code %d", exitCode)
	}
}
