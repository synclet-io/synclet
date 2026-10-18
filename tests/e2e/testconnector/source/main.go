package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	airbyte "github.com/synclet-io/synclet-go-sdk"
	"github.com/synclet-io/synclet-go-sdk/cli"
)

type sourceConfig struct {
	Streams           []streamConfig `json:"streams"`
	CrashAfterRecords int            `json:"crash_after_records"` // 0 = no crash
	ExitCode          int            `json:"exit_code"`           // exit code on crash
	EmitStateEvery    int            `json:"emit_state_every"`    // 0 = only final state
}

type streamConfig struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	RecordCount int    `json:"record_count"`
}

type testSource struct{}

func (s *testSource) Spec(context.Context) (*airbyte.ConnectorSpecification, error) {
	return &airbyte.ConnectorSpecification{
		DocumentationURL:    "https://github.com/synclet-io/synclet",
		SupportsIncremental: true,
		ConnectionSpecification: airbyte.ConnectionSpecification{
			Title:       "Test Source",
			Description: "A configurable test source for e2e testing",
			Type:        "object",
			Properties: airbyte.Properties{
				Properties: map[airbyte.PropertyName]airbyte.PropertySpec{
					"streams": {
						Description: "List of streams to emit",
					},
					"crash_after_records": {
						Description:  "Crash after emitting this many records (0 = no crash)",
						PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.Integer}},
					},
					"exit_code": {
						Description:  "Exit code on crash",
						PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.Integer}},
					},
					"emit_state_every": {
						Description:  "Emit state every N records (0 = only final state)",
						PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.Integer}},
					},
				},
			},
		},
	}, nil
}

func (s *testSource) Check(context.Context, sourceConfig) error {
	return nil
}

func (s *testSource) Discover(_ context.Context, cfg sourceConfig) (*airbyte.Catalog, error) {
	if len(cfg.Streams) == 0 {
		return &airbyte.Catalog{Streams: []airbyte.Stream{testStream("test_stream", "")}}, nil
	}
	streams := make([]airbyte.Stream, 0, len(cfg.Streams))
	for _, sc := range cfg.Streams {
		streams = append(streams, testStream(sc.Name, sc.Namespace))
	}
	return &airbyte.Catalog{Streams: streams}, nil
}

func testStream(name, namespace string) airbyte.Stream {
	return airbyte.Stream{
		Name:      name,
		Namespace: namespace,
		JSONSchema: airbyte.ObjectSchema(map[airbyte.PropertyName]airbyte.PropertySpec{
			"id":         {PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.Integer}}},
			"updated_at": {PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.Integer}}},
			"data":       {PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.String}}},
		}),
		SupportedSyncModes:  []airbyte.SyncMode{airbyte.SyncModeFullRefresh, airbyte.SyncModeIncremental},
		SourceDefinedCursor: true,
		DefaultCursorField:  []string{"updated_at"},
	}
}

type streamCursor struct {
	Cursor int `json:"cursor"`
}

func (s *testSource) Read(_ context.Context, cfg sourceConfig, _ *airbyte.ConfiguredCatalog, state *airbyte.InputState, em *airbyte.Emitter) error {
	totalEmitted := 0
	for _, stream := range cfg.Streams {
		desc := airbyte.StreamDescriptor{Name: stream.Name, Namespace: stream.Namespace}

		startCursor := 0
		if raw := state.Stream(desc); raw != nil {
			var prev streamCursor
			if err := json.Unmarshal(raw, &prev); err == nil {
				startCursor = prev.Cursor
			}
		}

		// Emit records from startCursor to startCursor + recordCount.
		for i := startCursor; i < startCursor+stream.RecordCount; i++ {
			record := map[string]any{
				"id":         i,
				"updated_at": i,
				"data":       fmt.Sprintf("record-%d", i),
			}

			if err := em.Record(desc, record); err != nil {
				return fmt.Errorf("emitting record: %w", err)
			}
			totalEmitted++

			// Crash simulation.
			if cfg.CrashAfterRecords > 0 && totalEmitted >= cfg.CrashAfterRecords {
				exitCode := cfg.ExitCode
				if exitCode == 0 {
					exitCode = 1
				}
				os.Exit(exitCode)
			}

			// Periodic state emission.
			if cfg.EmitStateEvery > 0 && totalEmitted%cfg.EmitStateEvery == 0 {
				if err := em.StreamState(desc, streamCursor{Cursor: i + 1}); err != nil {
					return fmt.Errorf("emitting state: %w", err)
				}
			}
		}

		if err := em.StreamState(desc, streamCursor{Cursor: startCursor + stream.RecordCount}); err != nil {
			return fmt.Errorf("emitting final state: %w", err)
		}
	}

	return nil
}

func main() {
	cli.RunSource(&testSource{})
}
