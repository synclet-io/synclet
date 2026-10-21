package notifyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// EmailChannel delivers notifications via email using the existing EmailSender.
type EmailChannel struct {
	emailSender EmailSender
}

// NewEmailChannel creates a new EmailChannel deliverer.
func NewEmailChannel(emailSender EmailSender) *EmailChannel {
	return &EmailChannel{emailSender: emailSender}
}

// Deliver sends a notification email to all configured recipients.
func (e *EmailChannel) Deliver(ctx context.Context, channel *NotificationChannel, event WebhookEvent) error {
	var config map[string]string
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("parsing channel config: %w", err)
	}

	recipientsStr, ok := config["recipients"]
	if !ok || recipientsStr == "" {
		return ErrRecipientsMissing
	}

	recipients := strings.Split(recipientsStr, ",")

	subject := fmt.Sprintf("[Synclet] Sync %s notification", event.Event)
	if event.WorkflowID != "" {
		subject = fmt.Sprintf("[Synclet] Workflow %s run failed", event.WorkflowName)
	}

	body := formatEmailBody(event)

	for _, recipient := range recipients {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}

		if err := e.emailSender.SendEmail(recipient, subject, body); err != nil {
			return fmt.Errorf("sending email to %s: %w", recipient, err)
		}
	}

	return nil
}

// formatEmailBody renders the event as HTML; every value from the event is
// escaped.
func formatEmailBody(event WebhookEvent) string {
	esc := html.EscapeString

	var builder strings.Builder
	if event.WorkflowID != "" {
		fmt.Fprintf(&builder, "<h2>Workflow Event: %s</h2>", esc(event.Event))
		fmt.Fprintf(&builder, "<p><strong>Workflow:</strong> %s (%s)</p>", esc(event.WorkflowName), esc(event.WorkflowID))
		fmt.Fprintf(&builder, "<p><strong>Run:</strong> %s</p>", esc(event.WorkflowRunID))
	} else {
		fmt.Fprintf(&builder, "<h2>Sync Event: %s</h2>", esc(event.Event))
	}

	if event.ConnectionID != "" {
		fmt.Fprintf(&builder, "<p><strong>Connection:</strong> %s</p>", esc(event.ConnectionID))
	}

	if event.JobID != "" {
		fmt.Fprintf(&builder, "<p><strong>Job:</strong> %s</p>", esc(event.JobID))
	}

	if event.Error != "" {
		fmt.Fprintf(&builder, "<p><strong>Error:</strong> %s</p>", esc(event.Error))
	}

	fmt.Fprintf(&builder, "<p><strong>Time:</strong> %s</p>", event.Timestamp.Format("2006-01-02 15:04:05 UTC"))

	return builder.String()
}
