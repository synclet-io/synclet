package notifyservice

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

var inviteEmailTemplate = template.Must(template.ParseFS(templateFS, "templates/invite_email.html"))

// InviteEmailParams holds the data for rendering an invite email.
type InviteEmailParams struct {
	WorkspaceName string
	InviterName   string
	Role          string
	AcceptURL     string
	ExpiresAt     time.Time
}

// RenderInviteEmail renders a branded HTML invite email and returns subject + body.
func RenderInviteEmail(params InviteEmailParams) (subject, htmlBody string, err error) {
	subject = fmt.Sprintf("You've been invited to join %s on Synclet", params.WorkspaceName)

	data := map[string]string{
		"WorkspaceName": params.WorkspaceName,
		"InviterName":   params.InviterName,
		"Role":          params.Role,
		"AcceptURL":     params.AcceptURL,
		"ExpiresAt":     params.ExpiresAt.Format("January 2, 2006"),
	}

	var buf bytes.Buffer
	if err := inviteEmailTemplate.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("executing invite email template: %w", err)
	}

	return subject, buf.String(), nil
}
