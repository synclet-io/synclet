package pipelineexec

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// RedactedMask replaces secret values in connector output.
const RedactedMask = "***"

const (
	minSecretLen       = 4
	minNestedSecretLen = 8
)

var secretKeySubstrings = []string{
	"password",
	"passphrase",
	"secret",
	"token",
	"credential",
	"private_key",
	"privatekey",
	"api_key",
	"apikey",
	"access_key",
	"accesskey",
	"client_key",
	"service_account",
	"hmac",
}

// Redactor masks secret configuration values in connector output. The zero
// value and nil redact nothing.
type Redactor struct {
	replacer *strings.Replacer
}

// NewRedactor collects secret values from connector configs: string values of
// at least minSecretLen bytes under secret-looking keys, strings of at least
// minNestedSecretLen bytes nested beneath such a key, and the same inside
// string values that hold a JSON object. Configs that are not JSON objects are
// skipped.
func NewRedactor(configs ...json.RawMessage) *Redactor {
	values := map[string]struct{}{}

	for _, raw := range configs {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		collectSecrets(obj, notSecret, values)
	}

	if len(values) == 0 {
		return &Redactor{}
	}

	secrets := make([]string, 0, len(values)*2)
	for value := range values {
		secrets = append(secrets, value)

		if escaped := jsonEscaped(value); escaped != value {
			secrets = append(secrets, escaped)
		}
	}

	// strings.Replacer prefers earlier arguments at the same position, so
	// longer secrets go first and a secret containing another is masked whole.
	sort.Slice(secrets, func(i, j int) bool {
		if len(secrets[i]) != len(secrets[j]) {
			return len(secrets[i]) > len(secrets[j])
		}

		return secrets[i] < secrets[j]
	})

	pairs := make([]string, 0, len(secrets)*2)
	for _, secret := range secrets {
		pairs = append(pairs, secret, RedactedMask)
	}

	return &Redactor{replacer: strings.NewReplacer(pairs...)}
}

// Redact returns s with every secret value replaced by RedactedMask.
func (r *Redactor) Redact(s string) string {
	if r == nil || r.replacer == nil {
		return s
	}

	return r.replacer.Replace(s)
}

type secretness int

const (
	notSecret secretness = iota
	nestedSecret
	directSecret
)

func collectSecrets(value any, level secretness, out map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			childLevel := notSecret

			switch {
			case isSecretKey(key):
				childLevel = directSecret
			case level != notSecret:
				childLevel = nestedSecret
			}

			collectSecrets(nested, childLevel, out)
		}
	case []any:
		for _, nested := range typed {
			collectSecrets(nested, level, out)
		}
	case string:
		addSecret(typed, level, out)
	}
}

func addSecret(value string, level secretness, out map[string]struct{}) {
	var nested map[string]any
	if strings.HasPrefix(strings.TrimSpace(value), "{") && json.Unmarshal([]byte(value), &nested) == nil {
		collectSecrets(nested, level, out)

		if level != notSecret {
			out[value] = struct{}{}
		}

		return
	}

	minLen := minNestedSecretLen
	if level == directSecret {
		minLen = minSecretLen
	}

	if level == notSecret || len(value) < minLen {
		return
	}

	out[value] = struct{}{}

	for line := range strings.SplitSeq(value, "\n") {
		if line = strings.TrimSpace(line); len(line) >= minNestedSecretLen {
			out[line] = struct{}{}
		}
	}
}

func isSecretKey(name string) bool {
	lower := strings.ToLower(name)
	for _, s := range secretKeySubstrings {
		if strings.Contains(lower, s) {
			return true
		}
	}

	return false
}

func jsonEscaped(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}

	return string(encoded[1 : len(encoded)-1])
}

// RedactHandler wraps handler so log lines and trace errors pass through
// redactor before handler sees them.
func RedactHandler(handler Handler, redactor *Redactor) Handler {
	if redactor == nil || redactor.replacer == nil {
		return handler
	}

	return &redactingHandler{Handler: handler, redactor: redactor}
}

type redactingHandler struct {
	Handler

	redactor *Redactor
}

func (h *redactingHandler) OnLog(ctx context.Context, line string) error {
	return h.Handler.OnLog(ctx, h.redactor.Redact(line))
}

func (h *redactingHandler) OnSourceTrace(ctx context.Context, msg *protocol.AirbyteTraceMessage) error {
	return h.Handler.OnSourceTrace(ctx, h.redactor.RedactTrace(msg))
}

// RedactTrace returns a copy of msg with its error texts redacted.
func (r *Redactor) RedactTrace(msg *protocol.AirbyteTraceMessage) *protocol.AirbyteTraceMessage {
	if msg == nil || msg.Error == nil || r == nil || r.replacer == nil {
		return msg
	}

	traceCopy := *msg
	errCopy := *msg.Error
	errCopy.Message = r.Redact(errCopy.Message)
	errCopy.InternalMessage = r.Redact(errCopy.InternalMessage)
	errCopy.StackTrace = r.Redact(errCopy.StackTrace)
	traceCopy.Error = &errCopy

	return &traceCopy
}
