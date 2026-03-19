package notifyservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// EmailSender sends emails. Pluggable interface per D-01.
type EmailSender interface {
	SendEmail(to, subject, htmlBody string) error
}

// SMTPConfig holds SMTP server configuration per D-03.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPEmailSender implements EmailSender using SMTP per D-01.
type SMTPEmailSender struct {
	config SMTPConfig
}

// NewSMTPEmailSender creates a new SMTPEmailSender.
func NewSMTPEmailSender(config SMTPConfig) *SMTPEmailSender {
	return &SMTPEmailSender{config: config}
}

// SendEmail sends an HTML email via SMTP over TLS.
func (s *SMTPEmailSender) SendEmail(to, subject, htmlBody string) error {
	addr := net.JoinHostPort(s.config.Host, fmt.Sprintf("%d", s.config.Port))
	msg := buildMIMEMessage(s.config.From, to, subject, htmlBody)

	// SECURITY: Use explicit TLS to prevent plaintext credential transmission.
	tlsConfig := &tls.Config{ServerName: s.config.Host}
	dialer := &tls.Dialer{Config: tlsConfig}
	conn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

func buildMIMEMessage(from, to, subject, htmlBody string) []byte {
	// Sanitize header values to prevent CRLF injection.
	sanitizer := strings.NewReplacer("\r", "", "\n", "")
	to = sanitizer.Replace(to)
	subject = sanitizer.Replace(subject)

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		from, to, subject,
	)

	return []byte(headers + htmlBody)
}

// NoOpEmailSender is used when SMTP is not configured. Silently discards emails.
type NoOpEmailSender struct{}

// NewNoOpEmailSender creates a new NoOpEmailSender.
func NewNoOpEmailSender() *NoOpEmailSender {
	return &NoOpEmailSender{}
}

// SendEmail is a no-op. SMTP not configured; configure SMTP_HOST env var for email delivery.
func (n *NoOpEmailSender) SendEmail(to, subject, htmlBody string) error {
	return nil
}
