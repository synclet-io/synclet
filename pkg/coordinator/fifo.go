package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	SourceStdoutFIFO = "source-stdout.fifo"
	DestStdinFIFO    = "dest-stdin.fifo"
	DestStdoutFIFO   = "dest-stdout.fifo"
)

// fifoMode lets connector containers open the FIFOs whatever user they run as,
// including root without CAP_DAC_OVERRIDE.
const fifoMode = 0o666

// CreateFIFOs creates the named pipes used for inter-container communication.
func CreateFIFOs(dir string) error {
	for _, name := range []string{SourceStdoutFIFO, DestStdinFIFO, DestStdoutFIFO} {
		path := filepath.Join(dir, name)
		if err := syscall.Mkfifo(path, fifoMode); err != nil {
			return fmt.Errorf("creating FIFO %s: %w", name, err)
		}

		if err := os.Chmod(path, fifoMode); err != nil { //nolint:gosec // the FIFOs are shared with connector containers running as other users
			return fmt.Errorf("setting mode of FIFO %s: %w", name, err)
		}
	}

	return nil
}

// WriteReadyFile writes the .ready sentinel file that signals connector containers to start.
func WriteReadyFile(dir string) error {
	return os.WriteFile(filepath.Join(dir, ".ready"), []byte("ready"), 0o644) //nolint:gosec // sentinel file, world-readable is fine
}

// ReadExitCode reads the exit code written by a connector's run script.
// A file that does not start with a digit holds no exit code yet and is an error.
func ReadExitCode(path string) (int32, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is constructed internally
	if err != nil {
		return -1, fmt.Errorf("reading exit code from %s: %w", path, err)
	}

	var (
		code   int32
		digits int
	)

	for _, b := range data {
		if b < '0' || b > '9' {
			break
		}

		code = code*10 + int32(b-'0')
		digits++
	}

	if digits == 0 {
		return -1, fmt.Errorf("no exit code in %s", path)
	}

	return code, nil
}

// WriteRunScript writes a shell script that runs a connector with FIFO redirects.
// It uses $AIRBYTE_ENTRYPOINT (set by Airbyte connector images) as the executable,
// falling back to running the command args directly if the env var is not set.
func WriteRunScript(scriptPath string, command []string, dataDir, stdinFIFO, stdoutFIFO, exitCodeFile string) error {
	var script strings.Builder

	script.WriteString("#!/bin/sh\n")
	// Airbyte connector images set AIRBYTE_ENTRYPOINT to the original Docker ENTRYPOINT.
	// Use it as the executable so that the connector args (check, read, write, etc.)
	// are passed as arguments to the correct binary.
	script.WriteString("${AIRBYTE_ENTRYPOINT:-} ")

	for _, arg := range command {
		script.WriteString(shellQuote(arg) + " ")
	}

	if stdinFIFO != "" {
		fmt.Fprintf(&script, "< %s ", shellQuote(filepath.Join(dataDir, stdinFIFO)))
	}

	// For FIFOs (sync mode), discard stderr to avoid corrupting Airbyte message stream.
	// For regular files (task mode), merge stderr into stdout for debugging visibility.
	if stdinFIFO != "" || strings.HasSuffix(stdoutFIFO, ".fifo") {
		fmt.Fprintf(&script, "> %s 2>/dev/null\n", shellQuote(filepath.Join(dataDir, stdoutFIFO)))
	} else {
		fmt.Fprintf(&script, "> %s 2>&1\n", shellQuote(filepath.Join(dataDir, stdoutFIFO)))
	}

	// The code goes to a temporary file first so that readers never see the file empty or partial.
	exitCodePath := filepath.Join(dataDir, exitCodeFile)
	fmt.Fprintf(&script, "echo $? > %s && mv %s %s\n", shellQuote(exitCodePath+".tmp"), shellQuote(exitCodePath+".tmp"), shellQuote(exitCodePath))

	return os.WriteFile(scriptPath, []byte(script.String()), 0o755) //nolint:gosec // run script needs execute permission
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
