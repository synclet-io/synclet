//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
)

const resetFrontendURL = "http://synclet.e2e.test"

var resetLinkPattern = regexp.MustCompile(`https?://[^\s"'<>]*/reset-password\?token=[^\s"'<>]+`)

// mailpit reads messages caught by a Mailpit instance (docker-compose.dev.yml).
// E2E_MAILPIT_URL and E2E_MAILPIT_SMTP_PORT point the test at a non-default one.
type mailpit struct {
	apiURL   string
	smtpPort string
}

func newMailpit(t *testing.T) *mailpit {
	t.Helper()

	m := &mailpit{apiURL: os.Getenv("E2E_MAILPIT_URL"), smtpPort: os.Getenv("E2E_MAILPIT_SMTP_PORT")}
	if m.apiURL == "" {
		m.apiURL = "http://localhost:8025"
	}
	if m.smtpPort == "" {
		m.smtpPort = "1025"
	}

	resp, err := http.Get(m.apiURL + "/api/v1/info")
	require.NoError(t, err, "Mailpit is not reachable at %s; start it with `docker compose -f docker-compose.dev.yml up -d mailpit`", m.apiURL)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "Mailpit info endpoint at %s", m.apiURL)

	return m
}

func (m *mailpit) getJSON(t *testing.T, path string, out any) {
	t.Helper()

	resp, err := http.Get(m.apiURL + path)
	require.NoError(t, err, "GET %s", path)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET %s", path)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(out), "decoding %s", path)
}

// waitForMessageHTML polls until a message addressed to `to` arrives and returns its HTML body.
func (m *mailpit) waitForMessageHTML(t *testing.T, to string, timeout time.Duration) string {
	t.Helper()

	query := "/api/v1/search?query=" + url.QueryEscape(fmt.Sprintf("to:%q", to))
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var found struct {
			Messages []struct {
				ID string `json:"ID"`
			} `json:"messages"`
		}
		m.getJSON(t, query, &found)

		if len(found.Messages) > 0 {
			var message struct {
				HTML string `json:"HTML"`
			}
			m.getJSON(t, "/api/v1/message/"+found.Messages[0].ID, &message)

			return message.HTML
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("no email to %s arrived in Mailpit within %s", to, timeout)

	return ""
}

// TestE2E_PasswordResetEmail requests a reset, reads the email from Mailpit
// and sets a new password with the link it carries.
func TestE2E_PasswordResetEmail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	mail := newMailpit(t)

	server := startSyncletServer(t, map[string]string{
		"NOTIFY_SMTP_HOST":       "localhost",
		"NOTIFY_SMTP_PORT":       mail.smtpPort,
		"NOTIFY_SMTP_TLS":        "none",
		"WORKSPACE_FRONTEND_URL": resetFrontendURL,
	})
	t.Cleanup(server.Stop)
	t.Cleanup(func() { truncateTestTables(t) })

	client := newAPIClient(server.baseURL)

	email := fmt.Sprintf("e2e-reset-%d@test.local", time.Now().UnixNano())
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword456!"

	_, err := client.auth.Register(ctx, connect.NewRequest(&authv1.RegisterRequest{
		Email:    email,
		Password: oldPassword,
		Name:     "E2E Reset User",
	}))
	require.NoError(t, err, "registering user")

	_, err = client.auth.RequestPasswordReset(ctx, connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: email}))
	require.NoError(t, err, "requesting password reset")

	body := mail.waitForMessageHTML(t, email, 30*time.Second)

	rawLink := resetLinkPattern.FindString(body)
	require.NotEmpty(t, rawLink, "reset link in email body:\n%s", body)

	link, err := url.Parse(html.UnescapeString(rawLink))
	require.NoError(t, err, "parsing reset link")
	require.Equal(t, resetFrontendURL, link.Scheme+"://"+link.Host, "reset link points at the frontend")

	token := link.Query().Get("token")
	require.NotEmpty(t, token, "token in reset link %s", link)

	_, err = client.auth.ResetPassword(ctx, connect.NewRequest(&authv1.ResetPasswordRequest{
		Token:       token,
		NewPassword: newPassword,
	}))
	require.NoError(t, err, "resetting password with the emailed token")

	_, err = client.auth.Login(ctx, connect.NewRequest(&authv1.LoginRequest{Email: email, Password: newPassword}))
	require.NoError(t, err, "logging in with the new password")

	_, err = client.auth.Login(ctx, connect.NewRequest(&authv1.LoginRequest{Email: email, Password: oldPassword}))
	require.Error(t, err, "old password must stop working")

	_, err = client.auth.ResetPassword(ctx, connect.NewRequest(&authv1.ResetPasswordRequest{
		Token:       token,
		NewPassword: "AnotherPassword789!",
	}))
	require.Error(t, err, "emailed token must work only once")
}
