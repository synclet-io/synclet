package main

import (
	"fmt"
	"os"

	airbyte "github.com/saturn4er/airbyte-go-sdk"
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

func (s *testSource) Spec(_ airbyte.LogTracker) (*airbyte.ConnectorSpecification, error) {
	return &airbyte.ConnectorSpecification{
		DocumentationURL:    "https://github.com/saturn4er/synclet",
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
						PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}},
					},
					"exit_code": {
						Description:  "Exit code on crash",
						PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}},
					},
					"emit_state_every": {
						Description:  "Emit state every N records (0 = only final state)",
						PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}},
					},
				},
			},
		},
	}, nil
}

func (s *testSource) Check(_ string, _ airbyte.LogTracker) error {
	return nil
}

func (s *testSource) Discover(srcCfgPath string, _ airbyte.LogTracker) (*airbyte.Catalog, error) {
	// Try to read config to discover streams dynamically.
	var cfg sourceConfig
	if srcCfgPath != "" {
		if err := airbyte.UnmarshalFromPath(srcCfgPath, &cfg); err == nil && len(cfg.Streams) > 0 {
			var streams []airbyte.Stream
			for _, sc := range cfg.Streams {
				streams = append(streams, airbyte.Stream{
					Name:      sc.Name,
					Namespace: sc.Namespace,
					JSONSchema: airbyte.Properties{
						Properties: map[airbyte.PropertyName]airbyte.PropertySpec{
							"id":         {PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}}},
							"updated_at": {PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}}},
							"data":       {PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.String}}},
						},
					},
					SupportedSyncModes:  []airbyte.SyncMode{airbyte.SyncModeFullRefresh, airbyte.SyncModeIncremental},
					SourceDefinedCursor: true,
					DefaultCursorField:  []string{"updated_at"},
				})
			}
			return &airbyte.Catalog{Streams: streams}, nil
		}
	}

	// Fallback: return a default test_stream.
	return &airbyte.Catalog{
		Streams: []airbyte.Stream{
			{
				Name: "test_stream",
				JSONSchema: airbyte.Properties{
					Properties: map[airbyte.PropertyName]airbyte.PropertySpec{
						"id":         {PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}}},
						"updated_at": {PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.Integer}}},
						"data":       {PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.String}}},
					},
				},
				SupportedSyncModes:  []airbyte.SyncMode{airbyte.SyncModeFullRefresh, airbyte.SyncModeIncremental},
				SourceDefinedCursor: true,
				DefaultCursorField:  []string{"updated_at"},
			},
		},
	}, nil
}

func (s *testSource) Read(srcCfgPath string, prevStatePath string, _ *airbyte.ConfiguredCatalog, tracker airbyte.MessageTracker) error {
	var cfg sourceConfig
	if err := airbyte.UnmarshalFromPath(srcCfgPath, &cfg); err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Load previous state for incremental syncs.
	prevStates := make(map[string]map[string]interface{})
	if prevStatePath != "" {
		loaded, err := airbyte.LoadStreamStates(prevStatePath)
		if err == nil {
			prevStates = loaded
		}
	}

	totalEmitted := 0
	for _, stream := range cfg.Streams {
		// Determine start cursor from previous state.
		startCursor := 0
		stateKey := stream.Name
		if stream.Namespace != "" {
			stateKey = stream.Namespace + ":" + stream.Name
		}

		if prev, ok := prevStates[stateKey]; ok {
			if cursor, ok := prev["cursor"]; ok {
				switch v := cursor.(type) {
				case float64:
					startCursor = int(v)
				case int:
					startCursor = v
				}
			}
		}

		// Emit records from startCursor to startCursor + recordCount.
		for i := startCursor; i < startCursor+stream.RecordCount; i++ {
			record := map[string]interface{}{
				"id":         i,
				"updated_at": i,
				"data":       fmt.Sprintf("record-%d", i),
			}

			if err := tracker.Record(record, stream.Name, stream.Namespace); err != nil {
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
				stateData := &airbyte.StreamState{
					StreamDescriptor: airbyte.StreamDescriptor{
						Name:      stream.Name,
						Namespace: strPtr(stream.Namespace),
					},
					StreamState: map[string]interface{}{
						"cursor": i + 1,
					},
				}
				if err := tracker.State(airbyte.StateTypeStream, stateData); err != nil {
					return fmt.Errorf("emitting state: %w", err)
				}
			}
		}

		// Emit final stream state.
		finalState := &airbyte.StreamState{
			StreamDescriptor: airbyte.StreamDescriptor{
				Name:      stream.Name,
				Namespace: strPtr(stream.Namespace),
			},
			StreamState: map[string]interface{}{
				"cursor": startCursor + stream.RecordCount,
			},
		}
		if err := tracker.State(airbyte.StateTypeStream, finalState); err != nil {
			return fmt.Errorf("emitting final state: %w", err)
		}
	}

	return nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func main() {
	if err := airbyte.NewSourceRunner(&testSource{}, os.Stdout).Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
