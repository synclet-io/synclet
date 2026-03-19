package notifyservice

// ValidationError represents an input validation error in the notify module.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error: " + e.Message
}
