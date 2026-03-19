package pipelinetasks

import "github.com/google/uuid"

// CreateTaskResult is returned by all Create*Task use cases.
type CreateTaskResult struct {
	TaskID uuid.UUID
}
