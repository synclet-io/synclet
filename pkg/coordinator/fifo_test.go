package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFIFOs_OpenableByAnyUser(t *testing.T) {
	oldMask := syscall.Umask(0o077)

	t.Cleanup(func() { syscall.Umask(oldMask) })

	dir := t.TempDir()
	require.NoError(t, CreateFIFOs(dir))

	for _, name := range []string{SourceStdoutFIFO, DestStdinFIFO, DestStdoutFIFO} {
		info, err := os.Stat(filepath.Join(dir, name))
		require.NoError(t, err)

		assert.Equal(t, os.ModeNamedPipe, info.Mode().Type(), name)
		assert.Equal(t, os.FileMode(0o666), info.Mode().Perm(), name)
	}
}

func TestReadExitCode_EmptyFileIsNotACode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.exitcode")
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	_, err := ReadExitCode(path)
	require.Error(t, err)
}

func TestReadExitCode_ReadsCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.exitcode")
	require.NoError(t, os.WriteFile(path, []byte("137\n"), 0o600))

	code, err := ReadExitCode(path)
	require.NoError(t, err)
	assert.Equal(t, int32(137), code)
}

func TestWaitForExitCode_WaitsForContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.exitcode")
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	go func() {
		time.Sleep(300 * time.Millisecond)

		_ = os.WriteFile(path, []byte("2\n"), 0o600)
	}()

	assert.Equal(t, int32(2), waitForExitCode(path, 5*time.Second))
}

func TestWaitForExitCode_NoDigitsTimesOutAsFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.exitcode")
	require.NoError(t, os.WriteFile(path, []byte("\n"), 0o600))

	assert.Equal(t, int32(-1), waitForExitCode(path, 500*time.Millisecond))
}

func TestWriteRunScript_WritesExitCodeAtomically(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	require.NoError(t, WriteRunScript(script, []string{"sh", "-c", "exit 3"}, dir, "", "out.log", "source.exitcode"))

	body, err := os.ReadFile(script) //nolint:gosec // test temp dir
	require.NoError(t, err)
	assert.Contains(t, string(body), "mv ")

	require.NoError(t, exec.CommandContext(t.Context(), "sh", script).Run()) //nolint:gosec // script written by the test

	code, err := ReadExitCode(filepath.Join(dir, "source.exitcode"))
	require.NoError(t, err)
	assert.Equal(t, int32(3), code)
	assert.NoFileExists(t, filepath.Join(dir, "source.exitcode.tmp"))
}
