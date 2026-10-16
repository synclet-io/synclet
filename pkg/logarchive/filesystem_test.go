package logarchive_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/logarchive"
)

const testKey = "logs/ws/conn/job/1.ndjson.gz"

func readTestKey(t *testing.T, archive interface {
	Open(ctx context.Context, key string) (io.ReadCloser, error)
},
) string {
	t.Helper()

	body, err := archive.Open(context.Background(), testKey)
	require.NoError(t, err)

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	require.NoError(t, body.Close())

	return string(data)
}

func TestFilesystem_RoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "archive")

	fs, err := logarchive.NewFilesystem(dir)
	require.NoError(t, err)

	require.NoError(t, fs.Put(ctx, testKey, strings.NewReader("first")))
	assert.Equal(t, "first", readTestKey(t, fs))

	require.NoError(t, fs.Put(ctx, testKey, strings.NewReader("second")))
	assert.Equal(t, "second", readTestKey(t, fs))

	entries, err := os.ReadDir(filepath.Join(dir, "logs", "ws", "conn", "job"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary files are left behind")
}

func TestFilesystem_OpenMissing(t *testing.T) {
	fs, err := logarchive.NewFilesystem(t.TempDir())
	require.NoError(t, err)

	_, err = fs.Open(context.Background(), "logs/missing.ndjson.gz")
	require.ErrorIs(t, err, logarchive.ErrNotFound)
}

func TestFilesystem_DeleteIsIdempotent(t *testing.T) {
	ctx := context.Background()

	fs, err := logarchive.NewFilesystem(t.TempDir())
	require.NoError(t, err)

	require.NoError(t, fs.Put(ctx, "a/1", strings.NewReader("x")))
	require.NoError(t, fs.Delete(ctx, "a/1", "a/2", "b/3"))
	require.NoError(t, fs.Delete(ctx, "a/1"))
	require.NoError(t, fs.Delete(ctx))

	_, err = fs.Open(ctx, "a/1")
	require.ErrorIs(t, err, logarchive.ErrNotFound)
}

func TestFilesystem_RejectsPathTraversal(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	dir := filepath.Join(base, "archive")

	fs, err := logarchive.NewFilesystem(dir)
	require.NoError(t, err)

	for _, key := range []string{"../escape", "logs/../../escape", "/etc/passwd", "logs//x", "logs/./x", `logs\..\x`, ""} {
		require.ErrorIs(t, fs.Put(ctx, key, strings.NewReader("x")), logarchive.ErrInvalidKey, key)

		_, err := fs.Open(ctx, key)
		require.ErrorIs(t, err, logarchive.ErrInvalidKey, key)
		require.ErrorIs(t, fs.Delete(ctx, key), logarchive.ErrInvalidKey, key)
	}

	_, err = os.Stat(filepath.Join(base, "escape"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestFilesystem_RejectsSymlinkEscape(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	dir := filepath.Join(base, "archive")
	outside := filepath.Join(base, "outside")

	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.MkdirAll(outside, 0o750))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "link")))

	fs, err := logarchive.NewFilesystem(dir)
	require.NoError(t, err)

	require.Error(t, fs.Put(ctx, "link/x", strings.NewReader("x")))

	entries, err := os.ReadDir(outside)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestNewFilesystem_RequiresDir(t *testing.T) {
	_, err := logarchive.NewFilesystem("")
	require.Error(t, err)
}

func TestFilesystem_MissingBaseDirectory(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "not-yet")

	fs, err := logarchive.NewFilesystem(dir)
	require.NoError(t, err)

	_, err = os.Stat(dir)
	require.ErrorIs(t, err, os.ErrNotExist, "nothing is created before the first write")

	_, err = fs.Open(ctx, "a/1")
	require.ErrorIs(t, err, logarchive.ErrNotFound)
	require.NoError(t, fs.Delete(ctx, "a/1"))
}

func TestNone(t *testing.T) {
	ctx := context.Background()

	require.ErrorIs(t, logarchive.None{}.Put(ctx, "k", strings.NewReader("x")), logarchive.ErrDisabled)

	_, err := logarchive.None{}.Open(ctx, "k")
	require.ErrorIs(t, err, logarchive.ErrDisabled)
	require.NoError(t, logarchive.None{}.Delete(ctx, "k"))
}
