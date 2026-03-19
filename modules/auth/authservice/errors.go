package authservice

// ValidationError represents an input validation error in the auth module.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error: " + e.Message
}
