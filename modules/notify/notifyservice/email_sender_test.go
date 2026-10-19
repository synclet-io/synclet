package notifyservice

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	eventTLSHello = "tls-hello"
	eventNoConn   = "no-conn"
)

// fakeSMTPServer accepts one connection and records what the client does: the SMTP
// verbs it sends, "data" once a message body arrives, and eventTLSHello when the
// client starts a TLS handshake. It ends the session at the first handshake.
func fakeSMTPServer(t *testing.T) (host string, port int, events <-chan []string) {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	result := make(chan []string, 1)

	go func() {
		_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(2 * time.Second))

		conn, err := listener.Accept()
		if err != nil {
			result <- []string{eventNoConn}

			return
		}

		defer func() { _ = conn.Close() }()

		result <- serveSMTP(conn)
	}()

	return addr.IP.String(), addr.Port, result
}

func serveSMTP(conn net.Conn) []string {
	var got []string

	reader := bufio.NewReader(conn)

	if startsTLS(conn, reader) {
		return append(got, eventTLSHello)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Write([]byte("220 fake ESMTP\r\n"))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return got
		}

		verb := strings.ToUpper(strings.Fields(line + " x")[0])
		got = append(got, verb)

		switch verb {
		case "EHLO":
			_, _ = conn.Write([]byte("250-fake\r\n250 STARTTLS\r\n"))
		case "STARTTLS":
			_, _ = conn.Write([]byte("220 ready\r\n"))

			if startsTLS(conn, reader) {
				got = append(got, eventTLSHello)
			}

			return got
		case "DATA":
			_, _ = conn.Write([]byte("354 go\r\n"))

			for {
				body, err := reader.ReadString('\n')
				if err != nil {
					return got
				}

				if body == ".\r\n" {
					break
				}
			}

			got = append(got, "data")
			_, _ = conn.Write([]byte("250 queued\r\n"))
		case "QUIT":
			_, _ = conn.Write([]byte("221 bye\r\n"))

			return got
		default:
			_, _ = conn.Write([]byte("250 ok\r\n"))
		}
	}
}

// startsTLS reports whether the client's next byte opens a TLS handshake record.
func startsTLS(conn net.Conn, reader *bufio.Reader) bool {
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	first, err := reader.Peek(1)

	return err == nil && first[0] == 0x16
}

func sendWithMode(t *testing.T, mode SMTPTLSMode) ([]string, error) {
	t.Helper()

	host, port, events := fakeSMTPServer(t)
	sender := NewSMTPEmailSender(SMTPConfig{Host: host, Port: port, From: "from@example.com", TLS: mode})

	sendErr := sender.SendEmail("to@example.com", "subject", "<p>body</p>")

	select {
	case got := <-events:
		return got, sendErr
	case <-time.After(5 * time.Second):
		t.Fatal("fake SMTP server did not finish")

		return nil, sendErr
	}
}

func TestSMTPEmailSender_ImplicitDialsTLS(t *testing.T) {
	got, err := sendWithMode(t, SMTPTLSImplicit)

	require.Error(t, err)
	assert.Equal(t, []string{eventTLSHello}, got)
}

func TestSMTPEmailSender_EmptyModeDialsTLS(t *testing.T) {
	got, err := sendWithMode(t, "")

	require.Error(t, err)
	assert.Equal(t, []string{eventTLSHello}, got)
}

func TestSMTPEmailSender_StartTLSUpgradesBeforeMail(t *testing.T) {
	got, err := sendWithMode(t, SMTPTLSStartTLS)

	require.Error(t, err)
	assert.Equal(t, []string{"EHLO", "STARTTLS", eventTLSHello}, got)
}

func TestSMTPEmailSender_NoneSendsPlainText(t *testing.T) {
	got, err := sendWithMode(t, SMTPTLSNone)

	require.NoError(t, err)
	assert.Equal(t, []string{"EHLO", "MAIL", "RCPT", "DATA", "data", "QUIT"}, got)
}

func TestSMTPEmailSender_UnknownModeRejectedBeforeDial(t *testing.T) {
	got, err := sendWithMode(t, "ssl")

	require.ErrorContains(t, err, `unknown tls mode "ssl"`)
	assert.Equal(t, []string{eventNoConn}, got)
}

func TestSMTPTLSMode_Valid(t *testing.T) {
	for _, mode := range []SMTPTLSMode{SMTPTLSImplicit, SMTPTLSStartTLS, SMTPTLSNone} {
		assert.True(t, mode.Valid(), strconv.Quote(string(mode)))
	}

	assert.False(t, SMTPTLSMode("ssl").Valid())
}
