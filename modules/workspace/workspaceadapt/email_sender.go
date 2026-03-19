package workspaceadapt

import (
	"context"
	"fmt"
	"time"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// EmailSenderAdapter implements workspaceservice.EmailSender using the notify module.
type EmailSenderAdapter struct {
	emailSender notifyservice.EmailSender
}

// NewEmailSenderAdapter creates a new EmailSenderAdapter.
func NewEmailSenderAdapter(emailSender notifyservice.EmailSender) *EmailSenderAdapter {
	return &EmailSenderAdapter{emailSender: emailSender}
}

// SendInviteEmail renders the invite email template and sends it.
func (a *EmailSenderAdapter) SendInviteEmail(_ context.Context, params workspaceservice.SendInviteEmailParams) error {
	expiresAt, _ := time.Parse("January 2, 2006", params.ExpiresAt)

	subject, htmlBody, err := notifyservice.RenderInviteEmail(notifyservice.InviteEmailParams{
		WorkspaceName: params.WorkspaceName,
		InviterName:   params.InviterName,
		Role:          params.Role,
		AcceptURL:     params.AcceptURL,
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		return fmt.Errorf("rendering invite email: %w", err)
	}

	if err := a.emailSender.SendEmail(params.To, subject, htmlBody); err != nil {
		return fmt.Errorf("sending invite email: %w", err)
	}

	return nil
}
