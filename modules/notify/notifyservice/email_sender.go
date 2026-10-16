package notifyservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	strconv "strconv"
	"strings"
)

// EmailSender sends emails. Pluggable interface per D-01.
type EmailSender interface {
	SendEmail(to, subject, htmlBody string) error
}

// SMTPTLSMode selects how the SMTP connection is encrypted.
type SMTPTLSMode string

const (
	// SMTPTLSImplicit opens the connection with TLS (usually port 465).
	SMTPTLSImplicit SMTPTLSMode = "implicit"
	// SMTPTLSStartTLS connects in plain text and upgrades with STARTTLS (usually port 587).
	SMTPTLSStartTLS SMTPTLSMode = "starttls"
	// SMTPTLSNone sends in plain text, for local catchers such as Mailpit.
	// net/smtp still refuses to send credentials unencrypted to a host other than localhost.
	SMTPTLSNone SMTPTLSMode = "none"
)

// Valid reports whether m is one of the known modes.
func (m SMTPTLSMode) Valid() bool {
	switch m {
	case SMTPTLSImplicit, SMTPTLSStartTLS, SMTPTLSNone:
		return true
	default:
		return false
	}
}

// SMTPConfig holds SMTP server configuration per D-03.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      SMTPTLSMode
}

// SMTPEmailSender implements EmailSender using SMTP per D-01.
type SMTPEmailSender struct {
	config SMTPConfig
}

// NewSMTPEmailSender creates a new SMTPEmailSender.
func NewSMTPEmailSender(config SMTPConfig) *SMTPEmailSender {
	return &SMTPEmailSender{config: config}
}

// SendEmail sends an HTML email via SMTP, encrypted as the TLS mode says.
func (s *SMTPEmailSender) SendEmail(to, subject, htmlBody string) error {
	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	msg := buildMIMEMessage(s.config.From, to, subject, htmlBody)
	tlsConfig := &tls.Config{ServerName: s.config.Host}

	var (
		conn net.Conn
		err  error
	)

	if s.config.TLS == SMTPTLSImplicit {
		conn, err = (&tls.Dialer{Config: tlsConfig}).DialContext(context.Background(), "tcp", addr)
	} else {
		conn, err = (&net.Dialer{}).DialContext(context.Background(), "tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("smtp dial (tls mode %s): %w", s.config.TLS, err)
	}

	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}

	defer func() { _ = client.Close() }()

	if s.config.TLS == SMTPTLSStartTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

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

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	if err := writer.Close(); err != nil {
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
