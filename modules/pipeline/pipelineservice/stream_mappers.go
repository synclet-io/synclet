package pipelineservice

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// CompileStreamMappers compiles a configured stream's mappers against schema,
// with the stream's selected fields, primary key and destination sync mode,
// and with salt as the hash salt.
// It returns an error when the mappers are not a JSON array of steps; compile
// problems are in the result's Errors.
func CompileStreamMappers(stream *protocol.ConfiguredAirbyteStream, schema json.RawMessage, salt []byte) (*mapper.Chain, *mapper.Result, error) {
	var steps []mapper.Step
	if err := json.Unmarshal(stream.Mappers, &steps); err != nil {
		return nil, nil, fmt.Errorf("decoding mappers: %w", err)
	}

	chain, result := mapper.Compile(mapper.Input{
		Schema:              schema,
		Steps:               steps,
		SelectedFields:      stream.SelectedFields,
		PrimaryKey:          stream.PrimaryKey,
		DestinationSyncMode: stream.DestinationSyncMode,
		Salt:                salt,
	})

	return chain, result, nil
}

// RenamedCursorField returns the stream's cursor field as the mappers' renames
// leave it, each rename step looked up once against the names before it.
func RenamedCursorField(stream *protocol.ConfiguredAirbyteStream) ([]string, error) {
	if len(stream.CursorField) == 0 {
		return stream.CursorField, nil
	}

	var steps []mapper.Step
	if err := json.Unmarshal(stream.Mappers, &steps); err != nil {
		return nil, fmt.Errorf("decoding mappers: %w", err)
	}

	cursor := append([]string(nil), stream.CursorField...)

	for _, step := range steps {
		if to, ok := step.Rename[cursor[0]]; ok {
			cursor[0] = to
		}
	}

	return cursor, nil
}

// HasMappers reports whether the stream has at least one mapper step.
func HasMappers(stream *protocol.ConfiguredAirbyteStream) bool {
	trimmed := strings.TrimSpace(string(stream.Mappers))

	return trimmed != "" && trimmed != "null" && trimmed != "[]"
}

// FormatMapperIssue describes one compile issue as "step <n>: <message>",
// with the field when the issue names one.
func FormatMapperIssue(issue mapper.Issue) string {
	prefix := fmt.Sprintf("step %d", issue.Step)
	if issue.Step == mapper.SelectionStep {
		prefix = "field selection"
	}

	if issue.Field != "" && !strings.Contains(issue.Message, fmt.Sprintf("%q", issue.Field)) {
		return fmt.Sprintf("%s, field %q: %s", prefix, issue.Field, issue.Message)
	}

	return fmt.Sprintf("%s: %s", prefix, issue.Message)
}

// FormatMapperIssues joins the issues with "; ".
func FormatMapperIssues(issues []mapper.Issue) string {
	parts := make([]string, len(issues))
	for i, issue := range issues {
		parts[i] = FormatMapperIssue(issue)
	}

	return strings.Join(parts, "; ")
}

// BuildSourceCatalog returns a copy of the catalog without the mappers, which
// only Synclet runs; everything else is as configured.
func BuildSourceCatalog(catalog *protocol.ConfiguredAirbyteCatalog) *protocol.ConfiguredAirbyteCatalog {
	src := &protocol.ConfiguredAirbyteCatalog{
		Streams: make([]protocol.ConfiguredAirbyteStream, len(catalog.Streams)),
	}

	for i, stream := range catalog.Streams {
		src.Streams[i] = stream
		src.Streams[i].Mappers = nil
	}

	return src
}
