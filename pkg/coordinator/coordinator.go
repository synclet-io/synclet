package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineroute"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// Config holds the configuration for the coordinator process.
// All sensitive config (source/dest configs, catalogs, state) is read from the
// mounted K8s Secret volume at SecretsDir. Non-sensitive metadata is passed via CLI flags.
type Config struct {
	JobID        string
	ConnectionID string
	ServerAddr   string
	DataDir      string
	SecretsDir   string // Path to mounted secrets volume (required)

	SourceID      string // UUID of source entity
	DestinationID string // UUID of destination entity

	SourceImage   string
	SourceCommand []string

	DestImage   string
	DestCommand []string

	// Namespace/prefix rewriting for the message router.
	NamespaceDefinition   string // "source", "destination", "custom"
	CustomNamespaceFormat string
	StreamPrefix          string
}

// Run executes the coordinator lifecycle inside a K8s pod:
// 1. Create FIFOs for inter-container communication
// 2. Write run scripts for source and destination containers
// 3. Signal connectors to start (touch .ready file)
// 4. Create Reporter for gRPC heartbeat/state/log delivery
// 5. Create K8sHandler for side effects
// 6. Run pipelineroute.Run() for message routing
// 7. Report completion via Reporter
//
// Config files (source-config.json, dest-config.json, catalogs, state) are mounted
// directly into connector containers via K8s Secret subPath mounts. The coordinator
// no longer copies files from /secrets to /shared.
//
// Per D-17, all config is received via CLI flags — no job dequeue logic.
func Run(ctx context.Context, cfg Config) error {
	slog.Info("coordinator: starting", "job_id", cfg.JobID, "connection_id", cfg.ConnectionID)

	// 1. Create FIFOs for inter-container communication.
	if err := CreateFIFOs(cfg.DataDir); err != nil {
		return fmt.Errorf("creating FIFOs: %w", err)
	}

	// 2. Write run scripts for source and destination containers.
	if err := WriteRunScript(
		filepath.Join(cfg.DataDir, "source-run.sh"),
		cfg.SourceCommand,
		cfg.DataDir,
		"",               // source reads from its own stdin (no FIFO)
		SourceStdoutFIFO, // source writes to FIFO
		"source.exitcode",
	); err != nil {
		return fmt.Errorf("writing source run script: %w", err)
	}

	if err := WriteRunScript(
		filepath.Join(cfg.DataDir, "dest-run.sh"),
		cfg.DestCommand,
		cfg.DataDir,
		DestStdinFIFO,  // dest reads from FIFO
		DestStdoutFIFO, // dest writes to FIFO
		"dest.exitcode",
	); err != nil {
		return fmt.Errorf("writing dest run script: %w", err)
	}

	// 3. Signal connectors to start.
	if err := WriteReadyFile(cfg.DataDir); err != nil {
		return fmt.Errorf("writing ready file: %w", err)
	}

	slog.Info("coordinator: connectors signaled, starting data pipeline")

	// 4. Create Reporter for gRPC heartbeat/state/log delivery.
	reporter := NewReporter(cfg.ServerAddr, cfg.JobID, cfg.ConnectionID)
	reporter.Start(ctx)
	defer reporter.Stop()

	// 5-6. Run the data pipeline with K8sHandler.
	stats, err := runPipeline(ctx, cfg, reporter)
	if stats == nil {
		stats = &pipelineroute.Stats{}
	}

	// Read exit codes from connector containers.
	// Connectors write exit code files after the process exits, which may happen
	// slightly after the FIFO streams close. Retry briefly to avoid false -1.
	sourceExit := waitForExitCode(filepath.Join(cfg.DataDir, "source.exitcode"), 10*time.Second)
	destExit := waitForExitCode(filepath.Join(cfg.DataDir, "dest.exitcode"), 10*time.Second)
	slog.Info("coordinator: connector exit codes", "job_id", cfg.JobID, "source_exit", sourceExit, "dest_exit", destExit)

	// 7. Report completion via Reporter.
	completionReq := &executorv1.ReportCompletionRequest{
		Success:        err == nil && sourceExit == 0 && destExit == 0,
		RecordsRead:    stats.RecordsRead,
		BytesSynced:    stats.BytesSynced,
		DurationMs:     stats.Duration.Milliseconds(),
		SourceExitCode: sourceExit,
		DestExitCode:   destExit,
	}

	if err != nil {
		completionReq.ErrorMessage = err.Error()
	} else if sourceExit != 0 {
		completionReq.ErrorMessage = fmt.Sprintf("source connector exited with code %d", sourceExit)
	} else if destExit != 0 {
		completionReq.ErrorMessage = fmt.Sprintf("destination connector exited with code %d", destExit)
	}

	reporter.QueueCompletion(completionReq)

	slog.Info("coordinator: completed", "job_id", cfg.JobID, "records_read", stats.RecordsRead)

	return nil
}

func runPipeline(ctx context.Context, cfg Config, reporter *Reporter) (*pipelineroute.Stats, error) {
	// Open FIFOs with context awareness. FIFO opens block until both ends connect;
	// if a connector fails to start, the open hangs indefinitely. Using a goroutine
	// with select on ctx.Done() ensures we unblock on context cancellation.
	sourceStdout, err := openFIFO(ctx, filepath.Join(cfg.DataDir, SourceStdoutFIFO), os.O_RDONLY)
	if err != nil {
		return nil, fmt.Errorf("opening source stdout FIFO: %w", err)
	}
	defer func() { _ = sourceStdout.Close() }()

	destStdin, err := openFIFO(ctx, filepath.Join(cfg.DataDir, DestStdinFIFO), os.O_WRONLY)
	if err != nil {
		return nil, fmt.Errorf("opening dest stdin FIFO: %w", err)
	}
	// Note: destStdin is an *os.File which implements io.WriteCloser.
	// pipelineroute.Run will close it when source EOF is reached.

	destStdout, err := openFIFO(ctx, filepath.Join(cfg.DataDir, DestStdoutFIFO), os.O_RDONLY)
	if err != nil {
		_ = destStdin.Close()
		return nil, fmt.Errorf("opening dest stdout FIFO: %w", err)
	}
	defer func() { _ = destStdout.Close() }()

	// Start heartbeat goroutine. During routing, intermediate stats are not
	// available (pipelineroute.Stats uses plain int64, not atomic), so heartbeats
	// send zeros. Final stats are sent in the completion report.
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				reporter.SendHeartbeat(heartbeatCtx, 0, 0)
			}
		}
	}()

	// Create K8s handler for side effects (state, config updates).
	handler := pipelineroute.NewK8sHandler(pipelineroute.K8sHandlerParams{
		Logger:        slog.Default(),
		Reporter:      reporter,
		SourceID:      cfg.SourceID,
		DestinationID: cfg.DestinationID,
	})

	// Build namespace rewriter from source catalog and connection settings.
	var rewriter *pipelineroute.NamespaceRewriter
	if cfg.NamespaceDefinition != "" {
		sourceCatalogBytes, readErr := os.ReadFile(filepath.Join(cfg.SecretsDir, "source-catalog"))
		if readErr != nil {
			slog.Warn("coordinator: could not read source catalog for rewriter", "error", readErr)
		} else {
			var catalog protocol.ConfiguredAirbyteCatalog
			if err := json.Unmarshal(sourceCatalogBytes, &catalog); err != nil {
				slog.Error("coordinator: failed to unmarshal source catalog for rewriter", "error", err)
			} else {
				nsDef := parseNamespaceDefinition(cfg.NamespaceDefinition)
				var customFmt *string
				if cfg.CustomNamespaceFormat != "" {
					customFmt = &cfg.CustomNamespaceFormat
				}
				var prefix *string
				if cfg.StreamPrefix != "" {
					prefix = &cfg.StreamPrefix
				}
				rewriter = pipelineroute.NewNamespaceRewriter(&catalog, nsDef, customFmt, prefix)
				slog.Info("coordinator: namespace rewriter created", "namespace_definition", cfg.NamespaceDefinition, "streams", len(catalog.Streams))
			}
		}
	}

	// Route messages using the shared router.
	return pipelineroute.Run(ctx, sourceStdout, destStdin, destStdout, handler, pipelineroute.RunConfig{
		Rewriter: rewriter,
	}, nil)
}

// waitForExitCode polls for an exit code file up to the given timeout.
// Connectors write the file after their process exits, which may lag behind
// the FIFO stream closing. Returns -1 only if the file never appears.
func waitForExitCode(path string, timeout time.Duration) int32 {
	deadline := time.Now().Add(timeout)
	for {
		code, err := ReadExitCode(path)
		if err == nil {
			return code
		}
		if time.Now().After(deadline) {
			slog.Warn("coordinator: exit code file not found after timeout", "path", path)
			return -1
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// openFIFO opens a FIFO with context cancellation support. FIFO opens block in
// the kernel until both ends connect, so we run OpenFile in a goroutine and
// select on ctx.Done() to avoid hanging indefinitely if a connector fails to start.
func openFIFO(ctx context.Context, path string, flag int) (*os.File, error) {
	type result struct {
		file *os.File
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		f, err := os.OpenFile(path, flag, 0)
		ch <- result{f, err}
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), ctx.Err())
	case r := <-ch:
		return r.file, r.err
	}
}

func parseNamespaceDefinition(s string) pipelineservice.NamespaceDefinition {
	switch s {
	case "Source", "source":
		return pipelineservice.NamespaceDefinitionSource
	case "Destination", "destination":
		return pipelineservice.NamespaceDefinitionDestination
	case "Custom", "custom":
		return pipelineservice.NamespaceDefinitionCustom
	default:
		return pipelineservice.NamespaceDefinitionSource
	}
}
