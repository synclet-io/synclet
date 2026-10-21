package coordinator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const (
	// connectorOutputTailSize bounds the connector output quoted in a task failure.
	connectorOutputTailSize = 4096
	// connectorOutputWindow is how much output before the tail is redacted
	// with it; it exceeds any config secret.
	connectorOutputWindow = 64 << 10
)

// connectorTaskTimeout bounds how long a connector task may run.
//
//nolint:gochecknoglobals // tests shorten it
var connectorTaskTimeout = 5 * time.Minute

// TaskConfig holds configuration for the task mode coordinator.
type TaskConfig struct {
	TaskID     string
	TaskType   string // "Check", "Spec", "Discover"
	ServerAddr string
	DataDir    string
	SecretsDir string

	redactor *pipelineexec.Redactor
}

// RunTask executes the task coordinator lifecycle inside a K8s pod:
// 1. Write connector run script to shared volume
// 2. Signal connector to start (.ready file)
// 3. Wait for connector to finish, read stdout log
// 4. Parse Airbyte messages, extract the relevant one based on task type
// 5. Report result via ReportConnectorTaskResult RPC
//
// Config file (config.json) is mounted directly into the connector container
// via K8s Secret subPath mount. The coordinator writes the run script and reads
// its own copy of the config only to mask config secrets in what it reports.
func RunTask(ctx context.Context, cfg TaskConfig) error {
	slog.Info("task coordinator: starting", "task_id", cfg.TaskID, "task_type", cfg.TaskType)

	cfg.redactor = taskConfigRedactor(cfg.SecretsDir)

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
	stdoutPath := filepath.Join(cfg.DataDir, "connector-stdout.log")

	if err := waitForFile(ctx, exitCodePath, connectorTaskTimeout); err != nil {
		err = fmt.Errorf("waiting for connector completion: %w", err)

		if tail := connectorOutputTail(stdoutPath, connectorOutputTailSize, cfg.redactor); tail != "" {
			slog.Warn("task coordinator: connector did not finish", "task_id", cfg.TaskID, "output_tail", tail)

			err = fmt.Errorf("%w; last connector output:\n%s", err, tail)
		}

		return reportTaskError(cfg, err)
	}

	exitCode := waitForExitCode(exitCodePath, 10*time.Second)
	slog.Info("task coordinator: connector finished", "task_id", cfg.TaskID, "exit_code", exitCode)

	// 4. Read connector stdout log and parse messages.
	slog.Debug("task coordinator: connector output tail", "task_id", cfg.TaskID,
		"output", connectorOutputTail(stdoutPath, connectorOutputTailSize, cfg.redactor))

	stdoutFile, err := os.Open(stdoutPath) //nolint:gosec // path is constructed internally
	if err != nil {
		return reportTaskError(cfg, fmt.Errorf("opening connector stdout: %w", err))
	}

	defer func() { _ = stdoutFile.Close() }()

	messages, err := readTaskMessages(stdoutFile)
	if err != nil {
		return reportTaskError(cfg, fmt.Errorf("reading connector output: %w", err))
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
	resultBytes, err := extractTaskResult(cfg.TaskType, messages, cfg.redactor)
	if err != nil {
		return reportTaskResult(cfg, false, err.Error(), nil)
	}

	slog.Info("task coordinator: result extracted", "task_id", cfg.TaskID, "task_type", cfg.TaskType)

	return reportTaskResult(cfg, true, "", resultBytes)
}

// readTaskMessages reads every message of a connector's output. A read error,
// including a message over protocol.TaskMessageMaxSize, ends the read.
func readTaskMessages(r io.Reader) ([]*protocol.AirbyteMessage, error) {
	reader := protocol.NewMessageReader(r, protocol.TaskMessageMaxSize)

	var messages []*protocol.AirbyteMessage

	for {
		msg, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return messages, nil
		}

		if err != nil {
			return messages, err //nolint:wrapcheck // the caller adds the context
		}

		messages = append(messages, msg)
	}
}

// connectorOutputTail returns up to size trailing bytes of the connector output
// at path with config secrets masked, starting at a line boundary when one
// falls inside the tail. It returns "" when the output is missing or empty.
//
// Redaction runs over connectorOutputWindow bytes before the tail as well, so
// a secret the tail would cut is masked whole. Output that does not end in a
// newline may stop inside a secret, so a trailing secret prefix is masked too.
func connectorOutputTail(path string, size int, redactor *pipelineexec.Redactor) string {
	file, err := os.Open(path) //nolint:gosec // path is constructed internally
	if err != nil {
		return ""
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}

	offset := max(info.Size()-int64(size+connectorOutputWindow), 0)

	data := make([]byte, info.Size()-offset)
	if _, err := file.ReadAt(data, offset); err != nil && !errors.Is(err, io.EOF) {
		return ""
	}

	text := string(data)
	if strings.HasSuffix(text, "\n") {
		text = redactor.Redact(text)
	} else {
		text = redactor.RedactTruncated(text)
	}

	if len(text) > size {
		text = text[len(text)-size:]
		if i := strings.IndexByte(text, '\n'); i >= 0 && i < len(text)-1 {
			text = text[i+1:]
		}
	}

	return strings.ToValidUTF8(strings.TrimSpace(text), "")
}

// extractTaskResult extracts the relevant Airbyte message based on task type
// and marshals it to JSON matching the domain result structs. A check status
// message is redacted, since connectors often echo the failing config value.
func extractTaskResult(taskType string, messages []*protocol.AirbyteMessage, redactor *pipelineexec.Redactor) ([]byte, error) {
	switch taskType {
	case "Check":
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeConnectionStatus && msg.ConnectionStatus != nil {
				result := struct {
					Success bool   `json:"Success"`
					Message string `json:"Message"`
				}{
					Success: msg.ConnectionStatus.Status == protocol.ConnectionStatusSucceeded,
					Message: redactor.Redact(msg.ConnectionStatus.Message),
				}

				return protocol.Marshal(result)
			}
		}

		return nil, errors.New("connector did not produce a CONNECTION_STATUS message")

	case "Spec":
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeSpec && msg.Spec != nil {
				specJSON, err := protocol.Marshal(msg.Spec)
				if err != nil {
					return nil, fmt.Errorf("marshaling spec: %w", err)
				}

				result := struct {
					Spec string `json:"Spec"`
				}{
					Spec: string(specJSON),
				}

				return protocol.Marshal(result)
			}
		}

		return nil, errors.New("connector did not produce a SPEC message")

	case "Discover":
		for _, msg := range messages {
			if msg.Type == protocol.MessageTypeCatalog && msg.Catalog != nil {
				catalogJSON, err := protocol.Marshal(msg.Catalog)
				if err != nil {
					return nil, fmt.Errorf("marshaling catalog: %w", err)
				}

				result := struct {
					Catalog string `json:"Catalog"`
				}{
					Catalog: string(catalogJSON),
				}

				return protocol.Marshal(result)
			}
		}

		return nil, errors.New("connector did not produce a CATALOG message")

	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

// retryPolicy bounds the retries of one report: exponential backoff from
// initial up to max, until total has passed since the first attempt.
type retryPolicy struct {
	initial time.Duration
	max     time.Duration
	total   time.Duration
}

// taskReportRetry outlasts a server restart and stays inside the server's
// default connector task running timeout (5m).
//
//nolint:gochecknoglobals // tests shorten it
var taskReportRetry = retryPolicy{initial: time.Second, max: 30 * time.Second, total: 4 * time.Minute}

// reportTaskResult sends the task result to the server via gRPC, with config
// secrets masked in the error message. A failure the server may recover from
// is retried; a rejected result is not.
func reportTaskResult(cfg TaskConfig, success bool, errMsg string, result []byte) error {
	clientOpts, err := clientOptions(cfg.SecretsDir)
	if err != nil {
		return err
	}

	client := executorv1connect.NewExecutorServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		cfg.ServerAddr,
		clientOpts...,
	)

	req := connect.NewRequest(&executorv1.ReportConnectorTaskResultRequest{
		TaskId:       cfg.TaskID,
		Success:      success,
		ErrorMessage: cfg.redactor.Redact(errMsg),
		Result:       result,
	})

	deadline := time.Now().Add(taskReportRetry.total)
	backoff := taskReportRetry.initial

	for attempt := 1; ; attempt++ {
		err = sendTaskResult(client, req)
		if err == nil {
			slog.Info("task coordinator: result reported", "task_id", cfg.TaskID, "success", success)

			return nil
		}

		if !retryableReport(err) || time.Now().Add(backoff).After(deadline) {
			slog.Error("task coordinator: failed to report result", "error", err, "task_id", cfg.TaskID, "attempts", attempt)

			return fmt.Errorf("reporting task result: %w", err)
		}

		slog.Warn("task coordinator: report failed, retrying", "error", err, "task_id", cfg.TaskID, "attempt", attempt, "backoff", backoff)
		time.Sleep(backoff)

		backoff = min(backoff*2, taskReportRetry.max)
	}
}

func sendTaskResult(client executorv1connect.ExecutorServiceClient, req *connect.Request[executorv1.ReportConnectorTaskResultRequest]) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.ReportConnectorTaskResult(ctx, req)

	return err //nolint:wrapcheck // wrapped once by reportTaskResult
}

// retryableReport reports whether a failed report may succeed when sent again:
// the server was unreachable, overloaded or restarting, not a rejection.
func retryableReport(err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeResourceExhausted,
		connect.CodeAborted, connect.CodeInternal, connect.CodeUnknown:
		return true
	default:
		return false
	}
}

// reportTaskError is a convenience function for reporting errors before result extraction.
func reportTaskError(cfg TaskConfig, err error) error {
	reportErr := reportTaskResult(cfg, false, err.Error(), nil)
	if reportErr != nil {
		return fmt.Errorf("%w (also failed to report: %w)", err, reportErr)
	}

	return err
}

// taskConfigRedactor masks the task config secrets mounted in secretsDir.
// Tasks without a config, or a config that cannot be read, redact nothing.
func taskConfigRedactor(secretsDir string) *pipelineexec.Redactor {
	if secretsDir == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(secretsDir, "task-config")) //nolint:gosec // fixed file name inside the mounted secrets volume.
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("task coordinator: could not read task config for redaction", "error", err)
		}

		return nil
	}

	return pipelineexec.NewRedactor(data)
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
