package pipelineexec

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

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
// value and nil redact nothing. It is safe for concurrent use.
type Redactor struct {
	mu       sync.RWMutex
	values   map[string]struct{}
	replacer *strings.Replacer
	secrets  []string
}

// NewRedactor collects secret values from connector configs: string values of
// at least minSecretLen bytes under secret-looking keys, strings of at least
// minNestedSecretLen bytes nested beneath such a key, and the same inside
// string values that hold a JSON object. Configs that are not JSON objects are
// skipped.
func NewRedactor(configs ...json.RawMessage) *Redactor {
	r := &Redactor{}
	r.Add(configs...)

	return r
}

// Add masks the secrets of configs as well, such as a config a connector
// replaced during a sync. It does nothing on a nil Redactor.
func (r *Redactor) Add(configs ...json.RawMessage) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.values == nil {
		r.values = map[string]struct{}{}
	}

	known := len(r.values)

	for _, raw := range configs {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		collectSecrets(obj, notSecret, r.values)
	}

	if len(r.values) == known {
		return
	}

	secrets := make([]string, 0, len(r.values)*2)
	for value := range r.values {
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

	r.replacer = strings.NewReplacer(pairs...)
	r.secrets = secrets
}

// Redact returns text with every secret value replaced by RedactedMask.
func (r *Redactor) Redact(text string) string {
	if r == nil {
		return text
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.replacer == nil {
		return text
	}

	return r.replacer.Replace(text)
}

// RedactTruncated is Redact for text cut off at its end, such as output still
// being written: it also masks a trailing prefix, of at least minSecretLen
// bytes, of a secret value.
func (r *Redactor) RedactTruncated(text string) string {
	text = r.Redact(text)
	if r == nil {
		return text
	}

	return r.maskTrailingPrefix(text)
}

// RedactHeadTruncated is Redact for text whose beginning was cut off, such as
// the newest part of a bounded buffer: it also masks a leading suffix, of at
// least minSecretLen bytes, of a secret value.
func (r *Redactor) RedactHeadTruncated(text string) string {
	text = r.Redact(text)
	if r == nil {
		return text
	}

	return r.maskLeadingSuffix(text)
}

func (r *Redactor) maskLeadingSuffix(text string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := 0

	for _, secret := range r.secrets {
		for n := min(len(secret)-1, len(text)); n >= minSecretLen; n-- {
			if strings.HasPrefix(text, secret[len(secret)-n:]) {
				start = max(start, n)

				break
			}
		}
	}

	if start == 0 {
		return text
	}

	return RedactedMask + text[start:]
}

func (r *Redactor) maskTrailingPrefix(text string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cut := len(text)

	for _, secret := range r.secrets {
		for n := min(len(secret)-1, len(text)); n >= minSecretLen; n-- {
			if strings.HasSuffix(text, secret[:n]) {
				cut = min(cut, len(text)-n)

				break
			}
		}
	}

	if cut == len(text) {
		return text
	}

	return text[:cut] + RedactedMask
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
// redactor before handler sees them, and log lines are cut to MaxLogLineSize
// once redacted. Secrets of a config a connector replaces through CONTROL are
// added to redactor. A nil redactor masks nothing.
func RedactHandler(handler Handler, redactor *Redactor) Handler {
	return &redactingHandler{Handler: handler, redactor: redactor}
}

type redactingHandler struct {
	Handler

	redactor *Redactor
}

func (h *redactingHandler) OnLog(ctx context.Context, line string) error {
	return h.Handler.OnLog(ctx, truncateLogLine(h.redactor.Redact(line)))
}

func (h *redactingHandler) OnSourceControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error {
	h.redactor.Add(controlConfig(msg)...)

	return h.Handler.OnSourceControl(ctx, msg)
}

func (h *redactingHandler) OnDestControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error {
	h.redactor.Add(controlConfig(msg)...)

	return h.Handler.OnDestControl(ctx, msg)
}

func controlConfig(msg *protocol.AirbyteControlMessage) []json.RawMessage {
	if msg == nil || msg.ConnectorConfig == nil {
		return nil
	}

	return []json.RawMessage{msg.ConnectorConfig.Config}
}

func (h *redactingHandler) OnSourceTrace(ctx context.Context, msg *protocol.AirbyteTraceMessage) error {
	return h.Handler.OnSourceTrace(ctx, h.redactor.RedactTrace(msg))
}

// RedactTrace returns a copy of msg with its error texts redacted.
func (r *Redactor) RedactTrace(msg *protocol.AirbyteTraceMessage) *protocol.AirbyteTraceMessage {
	if msg == nil || msg.Error == nil || r == nil {
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
