package notifyservice

type baseErr string

func (e baseErr) Error() string { return string(e) }

const (
	ErrNameRequired           baseErr = "name is required"
	ErrInvalidChannelType     baseErr = "invalid channel_type: must be one of slack, email, telegram"
	ErrWebhookURLRequired     baseErr = "webhook_url is required for slack channels"
	ErrRecipientsRequired     baseErr = "recipients is required for email channels"
	ErrBotTokenRequired       baseErr = "bot_token is required for telegram channels" //nolint:gosec // error message, not a credential
	ErrChatIDRequired         baseErr = "chat_id is required for telegram channels"
	ErrRecipientsMissing      baseErr = "recipients missing from channel config"
	ErrWebhookURLMissing      baseErr = "webhook_url missing from channel config"
	ErrBotTokenMissing        baseErr = "bot_token missing from channel config" //nolint:gosec // error message, not a credential
	ErrChatIDMissing          baseErr = "chat_id missing from channel config"
	ErrInvalidCondition       baseErr = "invalid condition: must be one of on_failure, on_consecutive_failures, on_zero_records"
	ErrConditionValueRequired baseErr = "condition_value must be >= 1 for on_consecutive_failures"
)

// ValidationError represents an input validation error in the notify module.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error: " + e.Message
}
