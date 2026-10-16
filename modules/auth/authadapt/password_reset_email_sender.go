package authadapt

import (
	"context"
	"fmt"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/notify/notifyservice"
)

// PasswordResetEmailSender implements authservice.PasswordResetEmailSender using the notify module.
type PasswordResetEmailSender struct {
	emailSender notifyservice.EmailSender
}

// NewPasswordResetEmailSender creates a new PasswordResetEmailSender.
func NewPasswordResetEmailSender(emailSender notifyservice.EmailSender) *PasswordResetEmailSender {
	return &PasswordResetEmailSender{emailSender: emailSender}
}

// SendPasswordResetEmail renders the password reset email template and sends it.
func (a *PasswordResetEmailSender) SendPasswordResetEmail(_ context.Context, params authservice.PasswordResetEmailParams) error {
	subject, htmlBody, err := notifyservice.RenderPasswordResetEmail(notifyservice.PasswordResetEmailParams{
		ResetURL: params.ResetURL,
	})
	if err != nil {
		return fmt.Errorf("rendering password reset email: %w", err)
	}

	if err := a.emailSender.SendEmail(params.To, subject, htmlBody); err != nil {
		return fmt.Errorf("sending password reset email: %w", err)
	}

	return nil
}
