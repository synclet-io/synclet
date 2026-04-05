package pipelineexec

import "github.com/google/uuid"

// NilIfEmpty returns nil if the string is empty, otherwise returns a pointer to the string.
func NilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

// GetWorkerID returns a generated ID for worker identification.
func GetWorkerID() string {
	return uuid.New().String()[:8]
}
