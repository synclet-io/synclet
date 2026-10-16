package notifyservice

import (
	"bytes"
	"fmt"
	"html/template"
)

var passwordResetEmailTemplate = template.Must(template.ParseFS(templateFS, "templates/password_reset_email.html"))

// PasswordResetEmailParams holds the data for rendering a password reset email.
type PasswordResetEmailParams struct {
	ResetURL string
}

// RenderPasswordResetEmail renders a branded HTML password reset email and returns subject + body.
func RenderPasswordResetEmail(params PasswordResetEmailParams) (subject, htmlBody string, err error) {
	subject = "Reset your Synclet password"

	var buf bytes.Buffer
	if err := passwordResetEmailTemplate.Execute(&buf, map[string]string{"ResetURL": params.ResetURL}); err != nil {
		return "", "", fmt.Errorf("executing password reset email template: %w", err)
	}

	return subject, buf.String(), nil
}
