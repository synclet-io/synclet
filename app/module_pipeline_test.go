package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"

	"github.com/synclet-io/synclet/pkg/logarchive"
)

type noopLogDelegate struct{}

func (noopLogDelegate) Info(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopLogDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopLogDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopLogDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopLogDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopLogDelegate) Named(string) logging.Delegate                      { return d }
func (d noopLogDelegate) SkipCallers(int) logging.Delegate                   { return d }

func buildLogArchive(t *testing.T, cfg *pipelineConfig) (pipelineservice.LogArchive, error) {
	t.Helper()

	lc := fxtest.NewLifecycle(t)
	archive, err := newPipelineLogArchive(lc, cfg, &logging.Logger{Delegate: noopLogDelegate{}})
	lc.RequireStart().RequireStop()

	return archive, err
}

func TestNewPipelineLogArchive(t *testing.T) {
	archive, err := buildLogArchive(t, &pipelineConfig{LogArchive: pipelineLogArchiveConfig{Backend: "none"}})
	require.NoError(t, err)
	assert.Equal(t, logarchive.None{}, archive)

	archive, err = buildLogArchive(t, &pipelineConfig{LogArchive: pipelineLogArchiveConfig{
		Backend: "filesystem",
		Dir:     filepath.Join(t.TempDir(), "logs"),
	}})
	require.NoError(t, err)
	assert.IsType(t, &logarchive.Filesystem{}, archive)

	archive, err = buildLogArchive(t, &pipelineConfig{LogArchive: pipelineLogArchiveConfig{
		Backend: "s3",
		S3:      pipelineLogArchiveS3Config{Bucket: "logs", Region: "auto", Endpoint: "http://localhost:9000"},
	}})
	require.NoError(t, err)
	assert.IsType(t, &logarchive.S3{}, archive)

	t.Setenv("STORAGE_EMULATOR_HOST", "localhost:4443")

	archive, err = buildLogArchive(t, &pipelineConfig{LogArchive: pipelineLogArchiveConfig{
		Backend: "gcs",
		GCS:     pipelineLogArchiveGCSConfig{Bucket: "logs"},
	}})
	require.NoError(t, err)
	assert.IsType(t, &logarchive.GCS{}, archive)
}

func TestNewPipelineLogArchive_NamesMisconfiguredVariable(t *testing.T) {
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")

	for name, tc := range map[string]struct {
		cfg     pipelineLogArchiveConfig
		mention string
	}{
		"unknown backend": {pipelineLogArchiveConfig{Backend: "gcs"}, "PIPELINE_LOG_ARCHIVE_BACKEND"},
		"filesystem dir":  {pipelineLogArchiveConfig{Backend: "filesystem"}, "PIPELINE_LOG_ARCHIVE_DIR"},
		"s3 bucket":       {pipelineLogArchiveConfig{Backend: "s3"}, "PIPELINE_LOG_ARCHIVE_S3_BUCKET"},
		"s3 keys": {
			pipelineLogArchiveConfig{Backend: "s3", S3: pipelineLogArchiveS3Config{Bucket: "b", Region: "r", AccessKeyID: "id"}},
			"PIPELINE_LOG_ARCHIVE_S3_SECRET_ACCESS_KEY",
		},
		"s3 region":  {pipelineLogArchiveConfig{Backend: "s3", S3: pipelineLogArchiveS3Config{Bucket: "b"}}, "PIPELINE_LOG_ARCHIVE_S3_REGION"},
		"gcs bucket": {pipelineLogArchiveConfig{Backend: "gcs"}, "PIPELINE_LOG_ARCHIVE_GCS_BUCKET"},
		"gcs credentials file": {
			pipelineLogArchiveConfig{Backend: "gcs", GCS: pipelineLogArchiveGCSConfig{Bucket: "b", CredentialsFile: "/nonexistent/key.json"}},
			"PIPELINE_LOG_ARCHIVE_GCS_CREDENTIALS_FILE",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildLogArchive(t, &pipelineConfig{LogArchive: tc.cfg})
			require.ErrorContains(t, err, tc.mention)
		})
	}
}

func TestNewPipelineServiceConfig_LogArchive(t *testing.T) {
	cfg := newPipelineServiceConfig(&pipelineConfig{MaxLogLinesPerAttempt: 7, LogArchive: pipelineLogArchiveConfig{Backend: "none"}})
	assert.False(t, cfg.LogArchiveEnabled)
	assert.Equal(t, 7, cfg.MaxLogLinesPerAttempt)

	cfg = newPipelineServiceConfig(&pipelineConfig{LogArchive: pipelineLogArchiveConfig{Backend: "s3"}})
	assert.True(t, cfg.LogArchiveEnabled)
}

func TestNewPipelineLogArchive_GCSRejectsInvalidKeyFile(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "key.json")
	require.NoError(t, os.WriteFile(keyFile, []byte(`{"type":"authorized_user"}`), 0o600))

	_, err := buildLogArchive(t, &pipelineConfig{LogArchive: pipelineLogArchiveConfig{
		Backend: "gcs",
		GCS:     pipelineLogArchiveGCSConfig{Bucket: "b", CredentialsFile: keyFile},
	}})
	require.ErrorContains(t, err, "PIPELINE_LOG_ARCHIVE_GCS")
}
