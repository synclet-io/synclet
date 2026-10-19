package app

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
)

func TestNotifySMTPDefaultsUseImplicitTLSOn465(t *testing.T) {
	unsetEnv(t, "NOTIFY_SMTP_PORT", "NOTIFY_SMTP_TLS")
	t.Setenv("NOTIFY_SMTP_HOST", "smtp.example.com")

	cfg, err := configutil.NewPrefixedConfigProvider[notifyConfig]("NOTIFY_")()
	require.NoError(t, err)

	assert.Equal(t, 465, cfg.SMTP.Port)
	assert.Equal(t, string(notifyservice.SMTPTLSImplicit), cfg.SMTP.TLS)
}

func TestNotifySMTPEmptyTLSBuildsSender(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	firstByte := make(chan byte, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			close(firstByte)

			return
		}

		defer func() { _ = conn.Close() }()

		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			close(firstByte)

			return
		}

		buf := make([]byte, 1)
		if _, err := io.ReadFull(conn, buf); err != nil {
			close(firstByte)

			return
		}

		firstByte <- buf[0]
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	cfg := &notifyConfig{SMTP: notifySMTPConfig{Host: "127.0.0.1", Port: port}}

	sender, err := newSMTPEmailSender(cfg, nil)
	require.NoError(t, err)
	require.IsType(t, &notifyservice.SMTPEmailSender{}, sender)

	require.Error(t, sender.SendEmail("to@example.com", "subject", "body"))

	// Implicit TLS opens with a TLS handshake record; plain SMTP waits for the server greeting.
	b, ok := <-firstByte
	require.True(t, ok, "sender wrote nothing to the server")
	assert.Equal(t, byte(0x16), b, "first byte is not a TLS handshake record")
}

func TestNotifySMTPUnknownTLSFailsStartup(t *testing.T) {
	cfg := &notifyConfig{SMTP: notifySMTPConfig{Host: "smtp.example.com", Port: 465, TLS: "ssl"}}

	_, err := newSMTPEmailSender(cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"ssl"`)
}
