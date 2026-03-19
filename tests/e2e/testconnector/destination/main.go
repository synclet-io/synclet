package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	airbyte "github.com/saturn4er/airbyte-go-sdk"
)

type destConfig struct {
	OutputDir string `json:"output_dir"`
}

type testDestination struct{}

func (d *testDestination) Spec(_ airbyte.LogTracker) (*airbyte.ConnectorSpecification, error) {
	return &airbyte.ConnectorSpecification{
		DocumentationURL: "https://github.com/saturn4er/synclet",
		SupportedDestinationSyncModes: []airbyte.DestinationSyncMode{
			airbyte.DestinationSyncModeAppend,
			airbyte.DestinationSyncModeOverwrite,
		},
		ConnectionSpecification: airbyte.ConnectionSpecification{
			Title:       "Test Destination",
			Description: "A file-writing test destination for e2e testing",
			Type:        "object",
			Properties: airbyte.Properties{
				Properties: map[airbyte.PropertyName]airbyte.PropertySpec{
					"output_dir": {
						Description:  "Directory to write output files",
						PropertyType: airbyte.PropertyType{Type: []airbyte.PropType{airbyte.String}},
					},
				},
			},
			Required: []airbyte.PropertyName{"output_dir"},
		},
	}, nil
}

func (d *testDestination) Check(dstCfgPath string, _ airbyte.LogTracker) error {
	var cfg destConfig
	if err := airbyte.UnmarshalFromPath(dstCfgPath, &cfg); err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	return os.MkdirAll(cfg.OutputDir, 0o755)
}

func (d *testDestination) Write(dstCfgPath string, _ string, inputReader io.Reader, tracker airbyte.MessageTracker) error {
	var cfg destConfig
	if err := airbyte.UnmarshalFromPath(dstCfgPath, &cfg); err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// Track per-stream record counts.
	streamCounts := make(map[string]int)
	// Track open file handles.
	files := make(map[string]*os.File)
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()

	scanner := bufio.NewScanner(inputReader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		line := scanner.Bytes()

		var msg struct {
			Type   string          `json:"type"`
			Record json.RawMessage `json:"record,omitempty"`
			State  json.RawMessage `json:"state,omitempty"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "RECORD":
			var rec struct {
				Stream    string          `json:"stream"`
				Namespace string          `json:"namespace"`
				Data      json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(msg.Record, &rec); err != nil {
				continue
			}

			// Determine output path.
			dir := cfg.OutputDir
			if rec.Namespace != "" {
				dir = filepath.Join(cfg.OutputDir, rec.Namespace)
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating namespace dir: %w", err)
			}

			filePath := filepath.Join(dir, rec.Stream+".jsonl")
			key := rec.Stream
			if rec.Namespace != "" {
				key = rec.Namespace + "/" + rec.Stream
			}

			f, ok := files[filePath]
			if !ok {
				var err error
				f, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
				if err != nil {
					return fmt.Errorf("opening output file: %w", err)
				}
				files[filePath] = f
			}

			if _, err := f.Write(rec.Data); err != nil {
				return fmt.Errorf("writing record: %w", err)
			}
			if _, err := f.WriteString("\n"); err != nil {
				return fmt.Errorf("writing newline: %w", err)
			}

			streamCounts[key]++

		case "STATE":
			// Re-emit state to confirm committed records.
			var stateMsg struct {
				Type   string          `json:"type"`
				Stream json.RawMessage `json:"stream,omitempty"`
				Global json.RawMessage `json:"global,omitempty"`
				Data   json.RawMessage `json:"data,omitempty"`
			}
			if err := json.Unmarshal(msg.State, &stateMsg); err != nil {
				continue
			}

			if stateMsg.Type == "STREAM" && stateMsg.Stream != nil {
				var streamState airbyte.StreamState
				if err := json.Unmarshal(stateMsg.Stream, &streamState); err != nil {
					continue
				}
				if err := tracker.State(airbyte.StateTypeStream, &streamState); err != nil {
					return fmt.Errorf("emitting state confirmation: %w", err)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	// Close all files before writing summary.
	for path, f := range files {
		f.Close()
		delete(files, path)
	}

	// Write summary.
	summaryPath := filepath.Join(cfg.OutputDir, "summary.json")
	summaryData, err := json.MarshalIndent(streamCounts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling summary: %w", err)
	}
	if err := os.WriteFile(summaryPath, summaryData, 0o644); err != nil {
		return fmt.Errorf("writing summary.json: %w", err)
	}

	return nil
}

func main() {
	if err := airbyte.NewDestinationRunner(&testDestination{}, os.Stdout).Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
