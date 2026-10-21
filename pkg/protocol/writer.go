package protocol

import (
	"bytes"
	"encoding/json"
	"io"
)

// MessageWriter writes AirbyteMessage structs as JSON lines to an io.Writer.
type MessageWriter struct {
	encoder *json.Encoder
}

// NewMessageWriter creates a MessageWriter that writes JSON lines to w.
// HTML escaping is disabled since protocol messages may contain HTML-like data.
func NewMessageWriter(w io.Writer) *MessageWriter {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	return &MessageWriter{encoder: encoder}
}

// Write serializes the message as a single JSON line followed by a newline.
func (mw *MessageWriter) Write(msg *AirbyteMessage) error {
	return mw.encoder.Encode(msg)
}

// Marshal encodes v like json.Marshal but leaves <, > and & unescaped, so a
// connector's output nested as a JSON string inside JSON grows at most about
// twofold.
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(v); err != nil {
		return nil, err //nolint:wrapcheck // same errors as json.Marshal
	}

	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
