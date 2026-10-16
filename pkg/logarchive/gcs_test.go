package logarchive_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/logarchive"
)

func TestNewGCS_Validation(t *testing.T) {
	ctx := context.Background()

	_, err := logarchive.NewGCS(ctx, logarchive.GCSConfig{})
	require.Error(t, err, "bucket is required")

	t.Setenv("STORAGE_EMULATOR_HOST", "localhost:4443")

	archive, err := logarchive.NewGCS(ctx, logarchive.GCSConfig{Bucket: "b"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, archive.Close()) })

	assert.Equal(t, "storage emulator at localhost:4443", archive.CredentialSource())
	require.ErrorIs(t, archive.Put(ctx, "../x", strings.NewReader("x")), logarchive.ErrInvalidKey)
	require.ErrorIs(t, archive.Delete(ctx, "/x"), logarchive.ErrInvalidKey)

	_, err = archive.Open(ctx, "")
	require.ErrorIs(t, err, logarchive.ErrInvalidKey)
}

// TestGCS_Integration runs against a real bucket (TEST_GCS_BUCKET with
// Application Default Credentials) or fake-gcs-server
// (STORAGE_EMULATOR_HOST=localhost:4443 with an existing TEST_GCS_BUCKET).
func TestGCS_Integration(t *testing.T) {
	bucket := os.Getenv("TEST_GCS_BUCKET")
	if bucket == "" {
		t.Skip("TEST_GCS_BUCKET is not set")
	}

	ctx := context.Background()

	archive, err := logarchive.NewGCS(ctx, logarchive.GCSConfig{
		Bucket:          bucket,
		Prefix:          "synclet-test/" + uuid.NewString(),
		CredentialsFile: os.Getenv("TEST_GCS_CREDENTIALS_FILE"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, archive.Close()) })

	require.NoError(t, archive.Put(ctx, testKey, strings.NewReader("hello")))
	assert.Equal(t, "hello", readTestKey(t, archive))

	require.NoError(t, archive.Put(ctx, testKey, strings.NewReader("again")))
	assert.Equal(t, "again", readTestKey(t, archive))

	require.Error(t, archive.Put(ctx, "logs/broken", failingReader{}))

	_, err = archive.Open(ctx, "logs/broken")
	require.ErrorIs(t, err, logarchive.ErrNotFound, "a failed upload leaves no object")

	require.NoError(t, archive.Delete(ctx, testKey, "logs/missing"))
	require.NoError(t, archive.Delete(ctx, testKey))

	_, err = archive.Open(ctx, testKey)
	require.ErrorIs(t, err, logarchive.ErrNotFound)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
