package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	airbyte "github.com/synclet-io/synclet-go-sdk"
	"github.com/synclet-io/synclet-go-sdk/cli"
)

type destConfig struct {
	OutputDir string `json:"output_dir"`
}

type testDestination struct{}

func (d *testDestination) Spec(context.Context) (*airbyte.ConnectorSpecification, error) {
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
						PropertyType: airbyte.PropertyType{Type: airbyte.PropTypes{airbyte.String}},
					},
				},
			},
			Required: []airbyte.PropertyName{"output_dir"},
		},
	}, nil
}

func (d *testDestination) Check(_ context.Context, cfg destConfig) error {
	return os.MkdirAll(cfg.OutputDir, 0o755)
}

func (d *testDestination) Write(_ context.Context, cfg destConfig, _ *airbyte.ConfiguredCatalog, in *airbyte.MessageReader, em *airbyte.Emitter) error {
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

	for {
		msg, err := in.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		switch msg.Type {
		case airbyte.MessageTypeRecord:
			rec := msg.Record
			if rec == nil {
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

		case airbyte.MessageTypeState:
			// Re-emit stream state to confirm committed records.
			if msg.State != nil && msg.State.Type == airbyte.StateTypeStream && msg.State.Stream != nil {
				if err := em.State(*msg.State); err != nil {
					return fmt.Errorf("emitting state confirmation: %w", err)
				}
			}
		}
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
	cli.RunDestination(&testDestination{})
}
