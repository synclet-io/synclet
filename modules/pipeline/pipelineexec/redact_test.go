package pipelineexec

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestRedactor(t *testing.T) {
	config := json.RawMessage(`{
		"host": "db.internal",
		"username": "synclet",
		"password": "hunter2-pass",
		"tunnel": {"ssh_private_key": "-----BEGIN KEY-----\nAAAABBBBCCCCDDDD\n-----END KEY-----"},
		"credentials": {"auth_type": "OAuth", "client_id": "client-abc-123", "refresh_token": "rt-xyz-987"},
		"service_account_info": "{\"type\":\"service_account\",\"private_key\":\"pk-\\\"quoted\\\"-value\"}",
		"short_secret": "ab",
		"port": 5432
	}`)
	dest := json.RawMessage(`{"api_key": "dst-key-555"}`)

	redactor := NewRedactor(config, dest, json.RawMessage(`not json`))

	tests := map[string]struct {
		in, out string
	}{
		"top-level secret":              {"login failed for password hunter2-pass", "login failed for password ***"},
		"non-secret values kept":        {"connecting to db.internal as synclet:5432", "connecting to db.internal as synclet:5432"},
		"nested under secret key":       {"client client-abc-123 refresh rt-xyz-987", "client *** refresh ***"},
		"multiline key line":            {"bad key line AAAABBBBCCCCDDDD", "bad key line ***"},
		"json string secret":            {`private_key pk-"quoted"-value`, "private_key ***"},
		"json-escaped echo":             {`{"password":"hunter2-pass","pk":"pk-\"quoted\"-value"}`, `{"password":"***","pk":"***"}`},
		"second config":                 {"key dst-key-555 rejected", "key *** rejected"},
		"short secret values not added": {"ab cd", "ab cd"},
		"short words under secret kept": {"OAuth flow", "OAuth flow"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.out, redactor.Redact(tt.in))
		})
	}
}

func TestRedactorNilAndEmpty(t *testing.T) {
	var nilRedactor *Redactor
	assert.Equal(t, "x", nilRedactor.Redact("x"))
	assert.Equal(t, "x", NewRedactor(json.RawMessage(`{"host":"x"}`)).Redact("x"))
}

type recordingHandler struct {
	DefaultHandler

	lines  []string
	traces []*protocol.AirbyteTraceMessage
}

func (h *recordingHandler) OnLog(_ context.Context, line string) error {
	h.lines = append(h.lines, line)

	return nil
}

func (h *recordingHandler) OnSourceTrace(_ context.Context, msg *protocol.AirbyteTraceMessage) error {
	h.traces = append(h.traces, msg)

	return nil
}

func TestRedactHandler(t *testing.T) {
	ctx := context.Background()
	inner := &recordingHandler{}
	handler := RedactHandler(inner, NewRedactor(json.RawMessage(`{"password":"hunter2-pass"}`)))

	require.NoError(t, handler.OnLog(ctx, formatLogLine("[src]", &protocol.AirbyteLogMessage{Level: "INFO", Message: "config password=hunter2-pass"})))

	original := &protocol.AirbyteTraceMessage{
		Type: protocol.TraceTypeError,
		Error: &protocol.AirbyteErrorTraceMessage{
			Message:         "auth failed hunter2-pass",
			InternalMessage: "pw hunter2-pass",
			StackTrace:      "at connect(hunter2-pass)",
		},
	}
	require.NoError(t, handler.OnSourceTrace(ctx, original))

	assert.Equal(t, []string{"[src] INFO: config password=***"}, inner.lines)
	require.Len(t, inner.traces, 1)
	assert.Equal(t, "auth failed ***", inner.traces[0].Error.Message)
	assert.Equal(t, "pw ***", inner.traces[0].Error.InternalMessage)
	assert.Equal(t, "at connect(***)", inner.traces[0].Error.StackTrace)
	assert.Equal(t, "auth failed hunter2-pass", original.Error.Message, "input message is not mutated")
}

func TestRedactor_RedactTruncated(t *testing.T) {
	redactor := NewRedactor(json.RawMessage(`{"password":"hunter2-pass"}`))

	tests := map[string]struct {
		in, out string
	}{
		"whole secret":               {"pw hunter2-pass", "pw ***"},
		"text cut inside a secret":   {"pw hunter2-p", "pw ***"},
		"cut before minSecretLen":    {"pw hun", "pw hun"},
		"secret then cut inside one": {"pw hunter2-pass and hunter", "pw *** and ***"},
		"no secret":                  {"plain text", "plain text"},
		"whole secret ends the text": {"hunter2-pass", "***"},
		"prefix not at the very end": {"hunter2- then more", "hunter2- then more"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.out, redactor.RedactTruncated(tt.in))
		})
	}

	var nilRedactor *Redactor
	assert.Equal(t, "hunter2-p", nilRedactor.RedactTruncated("hunter2-p"))
}

func TestRedactor_RedactHeadTruncated(t *testing.T) {
	redactor := NewRedactor(json.RawMessage(`{"password":"hunter2-pass"}`))

	tests := map[string]struct {
		in, out string
	}{
		"text starts inside a secret":  {"ter2-pass then more", "*** then more"},
		"suffix below minSecretLen":    {"ass then", "ass then"},
		"whole secret inside":          {"pw hunter2-pass ok", "pw *** ok"},
		"suffix not at the very start": {"x ter2-pass", "x ter2-pass"},
		"text ends inside a secret":    {"pw hunter2-p", "pw hunter2-p"},
		"no secret":                    {"plain text", "plain text"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.out, redactor.RedactHeadTruncated(tt.in))
		})
	}

	var nilRedactor *Redactor
	assert.Equal(t, "ter2-pass", nilRedactor.RedactHeadTruncated("ter2-pass"))
}

func TestRedactHandler_CutsLongLogLinesAfterRedaction(t *testing.T) {
	ctx := context.Background()
	long := strings.Repeat("x", MaxLogLineSize-4) + "hunter2-pass" + strings.Repeat("y", 1<<20)

	for name, redactor := range map[string]*Redactor{
		"with secrets":    NewRedactor(json.RawMessage(`{"password":"hunter2-pass"}`)),
		"without secrets": NewRedactor(),
	} {
		t.Run(name, func(t *testing.T) {
			inner := &recordingHandler{}
			require.NoError(t, RedactHandler(inner, redactor).OnLog(ctx, long))

			require.Len(t, inner.lines, 1)
			assert.Less(t, len(inner.lines[0]), MaxLogLineSize+64)
			assert.Contains(t, inner.lines[0], "bytes truncated")

			if redactor.replacer != nil {
				assert.NotContains(t, inner.lines[0], "hunt", "a secret at the cut is masked, not quoted in part")
			}
		})
	}
}
