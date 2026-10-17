package coordinator

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

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
