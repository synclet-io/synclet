package pipelineroute

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// NewFilteringReader wraps an io.Reader to filter RECORD message data
// by selected fields. Only streams with non-empty SelectedFields in the
// catalog are filtered; other messages pass through unchanged.
//
// The reader processes line-delimited JSON (Airbyte protocol format).
func NewFilteringReader(source io.Reader, catalog *protocol.ConfiguredAirbyteCatalog) io.Reader {
	// Build a map of stream key -> selected fields for quick lookup.
	fieldMap := make(map[string][]protocol.SelectedField)

	for _, stream := range catalog.Streams {
		if len(stream.SelectedFields) > 0 {
			key := stream.Stream.Name
			if stream.Stream.Namespace != "" {
				key = stream.Stream.Namespace + "." + stream.Stream.Name
			}

			fieldMap[key] = stream.SelectedFields
		}
	}

	if len(fieldMap) == 0 {
		return source // No filtering needed.
	}

	return &filteringReader{
		scanner:  bufio.NewScanner(source),
		fieldMap: fieldMap,
		buf:      bytes.Buffer{},
	}
}

// filteringReader implements io.Reader by reading line-delimited JSON from the
// source, filtering RECORD messages, and buffering output.
type filteringReader struct {
	scanner  *bufio.Scanner
	fieldMap map[string][]protocol.SelectedField
	buf      bytes.Buffer
	done     bool
}

// recordEnvelope is a minimal struct for detecting RECORD messages and
// extracting stream key + data without full deserialization.
type recordEnvelope struct {
	Type   string `json:"type"`
	Record *struct {
		Stream    string          `json:"stream"`
		Namespace string          `json:"namespace"`
		Data      json.RawMessage `json:"data"`
	} `json:"record"`
}

func (r *filteringReader) Read(buf []byte) (int, error) {
	for r.buf.Len() == 0 {
		if r.done {
			return 0, io.EOF
		}

		if !r.scanner.Scan() {
			r.done = true
			if err := r.scanner.Err(); err != nil {
				return 0, err
			}

			return 0, io.EOF
		}

		line := r.scanner.Bytes()
		filtered := r.filterLine(line)
		r.buf.Write(filtered)
		r.buf.WriteByte('\n')
	}

	return r.buf.Read(buf)
}

func (r *filteringReader) filterLine(line []byte) []byte {
	var env recordEnvelope
	if err := json.Unmarshal(line, &env); err != nil {
		return line // Not valid JSON; pass through.
	}

	if env.Type != "RECORD" || env.Record == nil {
		return line // Not a RECORD message; pass through.
	}

	key := env.Record.Stream
	if env.Record.Namespace != "" {
		key = env.Record.Namespace + "." + env.Record.Stream
	}

	fields, ok := r.fieldMap[key]
	if !ok {
		return line // Stream not in field map; pass through.
	}

	filteredData, err := pipelinecatalog.FilterRecordData(env.Record.Data, fields)
	if err != nil {
		return line // Filtering failed; pass through original.
	}

	// Rebuild the message with filtered data by doing a partial unmarshal/marshal.
	var full map[string]json.RawMessage
	if err := json.Unmarshal(line, &full); err != nil {
		return line
	}

	var record map[string]json.RawMessage
	if err := json.Unmarshal(full["record"], &record); err != nil {
		return line
	}

	record["data"] = filteredData

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return line
	}

	full["record"] = recordJSON

	result, err := json.Marshal(full)
	if err != nil {
		return line
	}

	return result
}
