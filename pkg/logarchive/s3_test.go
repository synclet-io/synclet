package logarchive_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/logarchive"
)

func TestNewS3_Validation(t *testing.T) {
	ctx := context.Background()

	_, err := logarchive.NewS3(ctx, logarchive.S3Config{Region: "us-east-1"})
	require.Error(t, err, "bucket is required")

	_, err = logarchive.NewS3(ctx, logarchive.S3Config{Region: "us-east-1", Bucket: "b", AccessKeyID: "id"})
	require.Error(t, err, "keys are set together")

	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")

	_, err = logarchive.NewS3(ctx, logarchive.S3Config{Bucket: "b"})
	require.ErrorIs(t, err, logarchive.ErrNoRegion)

	archive, err := logarchive.NewS3(ctx, logarchive.S3Config{Bucket: "b", Region: "auto", Endpoint: "http://localhost:9000", UsePathStyle: true})
	require.NoError(t, err)
	require.ErrorIs(t, archive.Put(ctx, "../x", strings.NewReader("x")), logarchive.ErrInvalidKey)
	require.ErrorIs(t, archive.Delete(ctx, "/x"), logarchive.ErrInvalidKey)

	_, err = archive.Open(ctx, "")
	require.ErrorIs(t, err, logarchive.ErrInvalidKey)
}

// TestS3_Integration runs against a real S3-compatible store, for example
// MinIO: TEST_S3_ENDPOINT=http://localhost:9000 TEST_S3_BUCKET=synclet-test
// TEST_S3_ACCESS_KEY_ID=minioadmin TEST_S3_SECRET_ACCESS_KEY=minioadmin.
func TestS3_Integration(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT is not set")
	}

	ctx := context.Background()

	region := os.Getenv("TEST_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	archive, err := logarchive.NewS3(ctx, logarchive.S3Config{
		Endpoint:        endpoint,
		Region:          region,
		Bucket:          os.Getenv("TEST_S3_BUCKET"),
		Prefix:          "synclet-test/" + uuid.NewString(),
		AccessKeyID:     os.Getenv("TEST_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("TEST_S3_SECRET_ACCESS_KEY"),
		UsePathStyle:    true,
	})
	require.NoError(t, err)

	require.NoError(t, archive.Put(ctx, testKey, strings.NewReader("hello")))
	assert.Equal(t, "hello", readTestKey(t, archive))

	require.NoError(t, archive.Put(ctx, testKey, nonSeeker{strings.NewReader("again")}))
	assert.Equal(t, "again", readTestKey(t, archive))

	require.NoError(t, archive.Delete(ctx, testKey, "logs/missing"))
	require.NoError(t, archive.Delete(ctx, testKey))

	_, err = archive.Open(ctx, testKey)
	require.ErrorIs(t, err, logarchive.ErrNotFound)
}

type nonSeeker struct {
	r *strings.Reader
}

func (n nonSeeker) Read(p []byte) (int, error) { return n.r.Read(p) }
