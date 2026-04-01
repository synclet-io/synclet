package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// TaskConfig holds configuration for the task mode coordinator.
type TaskConfig struct {
	TaskID     string
	TaskType   string // "Check", "Spec", "Discover"
	ServerAddr string
	DataDir    string
}

// RunTask executes the task coordinator lifecycle inside a K8s pod:
// 1. Write connector run script to shared volume
// 2. Signal connector to start (.ready file)
// 3. Wait for connector to finish, read stdout log
// 4. Parse Airbyte messages, extract the relevant one based on task type
// 5. Report result via ReportConnectorTaskResult RPC
//
// Config file (config.json) is mounted directly into the connector container
// via K8s Secret subPath mount. The coordinator only writes the run script.
func RunTask(ctx context.Context, cfg TaskConfig) error {
	slog.Info("task coordinator: starting", "task_id", cfg.TaskID, "task_type", cfg.TaskType)

	// 1. Write connector run script with the appropriate command args.
	connectorArgs := connectorTaskArgs(cfg.TaskType, cfg.DataDir)
	if err := WriteRunScript(
		filepath.Join(cfg.DataDir, "connector-run.sh"),
		connectorArgs,
		cfg.DataDir,
		"",                     // no stdin FIFO
		"connector-stdout.log", // capture stdout
		"connector.exitcode",
	); err != nil {
		return reportTaskError(cfg, fmt.Errorf("writing connector run script: %w", err))
	}

	// 2. Signal connector to start.
	if err := WriteReadyFile(cfg.DataDir); err != nil {
		return reportTaskError(cfg, fmt.Errorf("writing ready file: %w", err))
	}

	slog.Info("task coordinator: connector signaled")

	// 3. Wait for connector to finish by polling for exit code file.
	exitCodePath := filepath.Join(cfg.DataDir, "connector.exitcode")
	if err := waitForFile(ctx, exitCodePath, 5*time.Minute); err != nil {
		return reportTaskError(cfg, fmt.Errorf("waiting for connector completion: %w", err))
	}

	exitCode, _ := ReadExitCode(exitCodePath)
	slog.Info("task coordinator: connector finished", "task_id", cfg.TaskID, "exit_code", exitCode)

	// 4. Read connector stdout log and parse messages.
	stdoutPath := filepath.Join(cfg.DataDir, "connector-stdout.log")

	// Log raw connector output for debugging (especially useful for non-zero exit codes).
	if rawOutput, readErr := os.ReadFile(stdoutPath); readErr == nil { //nolint:gosec // path is constructed internally
		output := string(rawOutput)
		if len(output) > 4096 {
			output = output[:4096] + "... (truncated)"
		}

		slog.Debug("task coordinator: connector output", "task_id", cfg.TaskID, "output", output)
	} else {
		slog.Warn("task coordinator: could not read connector output", "task_id", cfg.TaskID, "error", readErr)
	}

	stdoutFile, err := os.Open(stdoutPath) //nolint:gosec // path is constructed internally
	if err != nil {
		return reportTaskError(cfg, fmt.Errorf("opening connector stdout: %w", err))
	}

	defer func() { _ = stdoutFile.Close() }()

	reader := protocol.NewMessageReader(stdoutFile)
	var messages []*protocol.AirbyteMessage

	for {
		msg, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			slog.Warn("task coordinator: skipping unparseable message", "error", err)

			continue
		}

		messages = append(messages, msg)
	}

	slog.Info("task coordinator: parsed messages", "task_id", cfg.TaskID, "count", len(messages))

	// Check for non-zero exit code.
	if exitCode != 0 {
		errMsg := fmt.Sprintf("connector exited with code %d", exitCode)
		// Check for TRACE error messages for a better error message.
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeTrace && msg.Trace != nil &&
				msg.Trace.Type == protocol.TraceTypeError && msg.Trace.Error != nil {
				errMsg = msg.Trace.Error.Message
				if msg.Trace.Error.InternalMessage != "" {
					errMsg = fmt.Sprintf("%s (internal: %s)", errMsg, msg.Trace.Error.InternalMessage)
				}

				break
			}
		}

		return reportTaskResult(cfg, false, errMsg, nil)
	}

	// 5. Extract the relevant message based on task type.
	resultBytes, err := extractTaskResult(cfg.TaskType, messages)
	if err != nil {
		return reportTaskResult(cfg, false, err.Error(), nil)
	}

	slog.Info("task coordinator: result extracted", "task_id", cfg.TaskID, "task_type", cfg.TaskType)

	return reportTaskResult(cfg, true, "", resultBytes)
}

// extractTaskResult extracts the relevant Airbyte message based on task type
// and marshals it to JSON matching the domain result structs.
func extractTaskResult(taskType string, messages []*protocol.AirbyteMessage) ([]byte, error) {
	switch taskType {
	case "Check":
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeConnectionStatus && msg.ConnectionStatus != nil {
				result := struct {
					Success bool   `json:"Success"`
					Message string `json:"Message"`
				}{
					Success: msg.ConnectionStatus.Status == protocol.ConnectionStatusSucceeded,
					Message: msg.ConnectionStatus.Message,
				}

				return json.Marshal(result)
			}
		}

		return nil, errors.New("connector did not produce a CONNECTION_STATUS message")

	case "Spec":
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeSpec && msg.Spec != nil {
				specJSON, err := json.Marshal(msg.Spec)
				if err != nil {
					return nil, fmt.Errorf("marshaling spec: %w", err)
				}

				result := struct {
					Spec string `json:"Spec"`
				}{
					Spec: string(specJSON),
				}

				return json.Marshal(result)
			}
		}

		return nil, errors.New("connector did not produce a SPEC message")

	case "Discover":
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeCatalog && msg.Catalog != nil {
				catalogJSON, err := json.Marshal(msg.Catalog)
				if err != nil {
					return nil, fmt.Errorf("marshaling catalog: %w", err)
				}

				result := struct {
					Catalog string `json:"Catalog"`
				}{
					Catalog: string(catalogJSON),
				}

				return json.Marshal(result)
			}
		}

		return nil, errors.New("connector did not produce a CATALOG message")

	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

// reportTaskResult sends the task result to the server via gRPC.
func reportTaskResult(cfg TaskConfig, success bool, errMsg string, result []byte) error {
	client := executorv1connect.NewExecutorServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		cfg.ServerAddr,
	)

	req := connect.NewRequest(&executorv1.ReportConnectorTaskResultRequest{
		TaskId:       cfg.TaskID,
		Success:      success,
		ErrorMessage: errMsg,
		Result:       result,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := client.ReportConnectorTaskResult(ctx, req); err != nil {
		slog.Error("task coordinator: failed to report result", "error", err, "task_id", cfg.TaskID)

		return fmt.Errorf("reporting task result: %w", err)
	}

	slog.Info("task coordinator: result reported", "task_id", cfg.TaskID, "success", success)

	return nil
}

// reportTaskError is a convenience function for reporting errors before result extraction.
func reportTaskError(cfg TaskConfig, err error) error {
	reportErr := reportTaskResult(cfg, false, err.Error(), nil)
	if reportErr != nil {
		return fmt.Errorf("%w (also failed to report: %w)", err, reportErr)
	}

	return err
}

// connectorTaskArgs returns the Airbyte CLI arguments for a connector task type.
func connectorTaskArgs(taskType, dataDir string) []string {
	switch taskType {
	case "Check":
		return []string{"check", "--config", filepath.Join(dataDir, "config.json")}
	case "Spec":
		return []string{"spec"}
	case "Discover":
		return []string{"discover", "--config", filepath.Join(dataDir, "config.json")}
	default:
		return []string{taskType}
	}
}

// waitForFile polls for a file to exist with a timeout.
func waitForFile(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.After(timeout)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timed out waiting for %s", path)
		case <-ticker.C:
			if _, err := os.Stat(path); err == nil {
				return nil
			}
		}
	}
}
