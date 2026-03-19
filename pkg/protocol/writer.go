package protocol

import (
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
