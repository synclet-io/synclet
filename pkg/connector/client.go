package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/synclet-io/synclet/pkg/container"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const pullTimeout = 10 * time.Minute

// ConnectorClient orchestrates Airbyte connector operations by composing
// the container runner with the protocol message reader/writer.
type ConnectorClient struct {
	runner      container.Runner
	memoryLimit int64
	cpuLimit    float64
}

// NewConnectorClient creates a new ConnectorClient backed by the given container runner.
func NewConnectorClient(runner container.Runner) *ConnectorClient {
	return &ConnectorClient{runner: runner}
}

// SetResourceLimits sets the memory and CPU limits applied to subsequent Read/Write calls.
func (c *ConnectorClient) SetResourceLimits(memoryLimit int64, cpuLimit float64) {
	c.memoryLimit = memoryLimit
	c.cpuLimit = cpuLimit
}

// Spec runs the connector's spec command and returns its specification.
// The container runs with network mode "none" since no external access is needed.
func (c *ConnectorClient) Spec(ctx context.Context, image string) (*protocol.ConnectorSpecification, error) {
	messages, err := c.runAndCollect(ctx, container.RunOptions{
		Image:       image,
		Command:     []string{"spec"},
		NetworkMode: "none",
	})
	if err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}

	for _, msg := range messages {
		if msg.Type == protocol.MessageTypeSpec && msg.Spec != nil {
			return msg.Spec, nil
		}
	}

	return nil, errors.New("spec: connector did not produce a SPEC message")
}

// Check validates a connector configuration by running the check command.
func (c *ConnectorClient) Check(ctx context.Context, image string, config json.RawMessage) (*protocol.AirbyteConnectionStatus, error) {
	messages, err := c.runAndCollect(ctx, container.RunOptions{
		Image:       image,
		Command:     []string{"check", "--config", "/tmp/config.json"},
		ConfigFile:  config,
		NetworkMode: "bridge",
	})
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}

	for _, msg := range messages {
		if msg.Type == protocol.MessageTypeConnectionStatus && msg.ConnectionStatus != nil {
			return msg.ConnectionStatus, nil
		}
	}

	return nil, errors.New("check: connector did not produce a CONNECTION_STATUS message")
}

// Discover discovers the catalog (available streams) for a connector.
func (c *ConnectorClient) Discover(ctx context.Context, image string, config json.RawMessage) (*protocol.AirbyteCatalog, error) {
	messages, err := c.runAndCollect(ctx, container.RunOptions{
		Image:       image,
		Command:     []string{"discover", "--config", "/tmp/config.json"},
		ConfigFile:  config,
		NetworkMode: "bridge",
	})
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	for _, msg := range messages {
		if msg.Type == protocol.MessageTypeCatalog && msg.Catalog != nil {
			return msg.Catalog, nil
		}
	}

	return nil, errors.New("discover: connector did not produce a CATALOG message")
}

// Read starts a source connector and returns its stdout for the caller to read messages from.
// The caller must close the returned ReadCloser and call the cleanup function when done.
// When the connector exits with a non-zero code, the returned reader surfaces the error
// after all buffered messages are consumed (at the point where it would normally return EOF).
func (c *ConnectorClient) Read(ctx context.Context, image string, config json.RawMessage, catalog *protocol.ConfiguredAirbyteCatalog, state json.RawMessage, labels map[string]string) (io.ReadCloser, func(), error) {
	catalogBytes, err := json.Marshal(catalog)
	if err != nil {
		return nil, nil, fmt.Errorf("read: marshaling catalog: %w", err)
	}

	cmd := []string{"read", "--config", "/tmp/config.json", "--catalog", "/tmp/catalog.json"}

	var stateBytes []byte
	if state != nil {
		stateBytes = state

		cmd = append(cmd, "--state", "/tmp/state.json")
	}

	result, err := c.runWithAutoPull(ctx, container.RunOptions{
		Image:       image,
		Command:     cmd,
		ConfigFile:  config,
		CatalogFile: catalogBytes,
		StateFile:   stateBytes,
		MemoryLimit: c.memoryLimit,
		CPULimit:    c.cpuLimit,
		NetworkMode: "bridge",
		Labels:      labels,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("read: %w", err)
	}

	cleanup := func() {
		// Stop with 30s grace period (SIGTERM -> wait 30s -> SIGKILL) before removing.
		// This allows the connector to flush final STATE messages to stdout.
		_ = c.runner.StopWithTimeout(context.Background(), result.ContainerID, 30)
		_ = c.runner.Remove(context.Background(), result.ContainerID)
	}

	wrapped := newExitAwareReader(result.Stdout, result.Stderr, result.Done, &result.ExitCode, "source")

	return wrapped, cleanup, nil
}

// Write starts a destination connector with the given stdin reader and returns its stdout.
// The caller pipes source messages into stdin and reads dest output from the returned ReadCloser.
// The caller must close the returned ReadCloser and call the cleanup function when done.
// When the connector exits with a non-zero code, the returned reader surfaces the error
// after all buffered messages are consumed.
func (c *ConnectorClient) Write(ctx context.Context, image string, config json.RawMessage, catalog *protocol.ConfiguredAirbyteCatalog, stdin io.Reader, labels map[string]string) (io.ReadCloser, func(), error) {
	catalogBytes, err := json.Marshal(catalog)
	if err != nil {
		return nil, nil, fmt.Errorf("write: marshaling catalog: %w", err)
	}

	result, err := c.runWithAutoPull(ctx, container.RunOptions{
		Image:       image,
		Command:     []string{"write", "--config", "/tmp/config.json", "--catalog", "/tmp/catalog.json"},
		ConfigFile:  config,
		CatalogFile: catalogBytes,
		Stdin:       stdin,
		MemoryLimit: c.memoryLimit,
		CPULimit:    c.cpuLimit,
		NetworkMode: "bridge",
		Labels:      labels,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("write: %w", err)
	}

	cleanup := func() {
		// Stop with 30s grace period (SIGTERM -> wait 30s -> SIGKILL) before removing.
		// This allows the connector to flush final STATE messages to stdout.
		_ = c.runner.StopWithTimeout(context.Background(), result.ContainerID, 30)
		_ = c.runner.Remove(context.Background(), result.ContainerID)
	}

	wrapped := newExitAwareReader(result.Stdout, result.Stderr, result.Done, &result.ExitCode, "destination")

	return wrapped, cleanup, nil
}

// exitAwareReader wraps a container's stdout. When the underlying reader returns
// EOF, it waits for the container to exit and checks the exit code. If the container
// exited with a non-zero code, it returns an error instead of EOF so the caller
// knows the connector failed.
type exitAwareReader struct {
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	done     <-chan struct{}
	exitCode *int
	role     string // "source" or "destination", for error messages
}

func newExitAwareReader(stdout, stderr io.ReadCloser, done <-chan struct{}, exitCode *int, role string) *exitAwareReader {
	return &exitAwareReader{
		stdout:   stdout,
		stderr:   stderr,
		done:     done,
		exitCode: exitCode,
		role:     role,
	}
}

func (r *exitAwareReader) Read(p []byte) (int, error) {
	n, err := r.stdout.Read(p)
	if err == io.EOF {
		// Stdout closed — wait for container to fully exit.
		<-r.done

		if *r.exitCode != 0 {
			stderrContent, _ := io.ReadAll(r.stderr)
			stderrStr := truncateString(string(bytes.TrimSpace(stderrContent)), 1024)

			return n, &ExitCodeError{
				ExitCode: *r.exitCode,
				Role:     r.role,
				Stderr:   stderrStr,
			}
		}
	}

	return n, err
}

func (r *exitAwareReader) Close() error {
	return r.stdout.Close()
}

// runWithAutoPull attempts runner.Run, and if it fails with an image-not-found
// error, pulls the image and retries once. Per D-06: try-run first, pull on failure, retry.
func (c *ConnectorClient) runWithAutoPull(ctx context.Context, opts container.RunOptions) (*container.RunResult, error) {
	result, err := c.runner.Run(ctx, opts)
	if err == nil {
		return result, nil
	}

	// Check if the error indicates a missing image.
	// Docker API returns "No such image" in the error message when the image is not found.
	if !isImageNotFoundError(err) {
		return nil, err
	}

	// Pull the image with a 10-minute timeout (per D-07).
	pullCtx, cancel := context.WithTimeout(ctx, pullTimeout)
	defer cancel()

	if pullErr := c.runner.Pull(pullCtx, opts.Image); pullErr != nil {
		// Per D-10: no retries on pull failure, fail immediately with clear message.
		return nil, fmt.Errorf("image %q not found locally and pull failed: %w", opts.Image, pullErr)
	}

	// Pin by digest to prevent TOCTOU attacks (SEC-13).
	if digest, digestErr := c.runner.ResolveDigest(ctx, opts.Image); digestErr == nil {
		opts.Image = digest
	}

	// Retry the run after successful pull.
	return c.runner.Run(ctx, opts)
}

// isImageNotFoundError checks if an error is a Docker "image not found" error.
func isImageNotFoundError(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "No such image") || strings.Contains(msg, "reference does not exist")
}

// runAndCollect runs a container, reads all messages from STDOUT until the container exits,
// and returns them. If the container emits a TRACE error message, it is returned as an error.
// If the container exits with a non-zero code and no TRACE error was found, a generic error
// with stderr content is returned.
func (c *ConnectorClient) runAndCollect(ctx context.Context, opts container.RunOptions) ([]*protocol.AirbyteMessage, error) {
	result, err := c.runWithAutoPull(ctx, opts)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = c.runner.Remove(context.Background(), result.ContainerID)
	}()

	// Read all messages from STDOUT.
	reader := protocol.NewMessageReader(result.Stdout)
	var messages []*protocol.AirbyteMessage

	for {
		msg, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("reading messages: %w", err)
		}

		messages = append(messages, msg)
	}

	// Wait for the container to exit.
	<-result.Done

	// Check for TRACE error messages.
	if traceErr := extractTraceError(messages); traceErr != nil {
		return nil, traceErr
	}

	// If the container exited with a non-zero code, return an error with stderr content.
	if result.ExitCode != 0 {
		stderrContent, _ := io.ReadAll(result.Stderr)
		stderrStr := truncateString(string(bytes.TrimSpace(stderrContent)), 1024)

		return nil, fmt.Errorf("container exited with code %d: %s", result.ExitCode, stderrStr)
	}

	return messages, nil
}

// extractTraceError looks through messages for a TRACE error and returns it as an error.
func extractTraceError(messages []*protocol.AirbyteMessage) error {
	for _, msg := range messages {
		if msg.Type == protocol.MessageTypeTrace && msg.Trace != nil && msg.Trace.Type == protocol.TraceTypeError && msg.Trace.Error != nil {
			errMsg := msg.Trace.Error.Message
			if msg.Trace.Error.InternalMessage != "" {
				errMsg = fmt.Sprintf("%s (internal: %s)", errMsg, msg.Trace.Error.InternalMessage)
			}

			return &connectorError{
				Message:     errMsg,
				FailureType: msg.Trace.Error.FailureType,
			}
		}
	}

	return nil
}

// truncateString truncates a string to maxLen, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "..."
}
