package notifyservice

import (
	"context"
	"encoding/json"
	"fmt"
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
		return fmt.Errorf("recipients missing from channel config")
	}

	recipients := strings.Split(recipientsStr, ",")
	subject := fmt.Sprintf("[Synclet] Sync %s notification", event.Event)
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

func formatEmailBody(event WebhookEvent) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<h2>Sync Event: %s</h2>", event.Event)
	if event.ConnectionID != "" {
		fmt.Fprintf(&sb, "<p><strong>Connection:</strong> %s</p>", event.ConnectionID)
	}
	if event.JobID != "" {
		fmt.Fprintf(&sb, "<p><strong>Job:</strong> %s</p>", event.JobID)
	}
	if event.Error != "" {
		fmt.Fprintf(&sb, "<p><strong>Error:</strong> %s</p>", event.Error)
	}
	fmt.Fprintf(&sb, "<p><strong>Time:</strong> %s</p>", event.Timestamp.Format("2006-01-02 15:04:05 UTC"))
	return sb.String()
}
