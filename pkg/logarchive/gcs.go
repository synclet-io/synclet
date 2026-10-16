package logarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
)

const gcsDeleteConcurrency = 8

// GCSConfig configures the GCS backend. Without CredentialsFile the client
// uses Application Default Credentials: the GKE metadata server under Workload
// Identity, GOOGLE_APPLICATION_CREDENTIALS, or gcloud ADC locally.
type GCSConfig struct {
	Bucket          string
	Prefix          string
	CredentialsFile string
}

// GCS stores objects in a Google Cloud Storage bucket.
type GCS struct {
	client           *storage.Client
	bucket           *storage.BucketHandle
	prefix           string
	credentialSource string
}

// NewGCS builds the GCS client. It does not contact the bucket.
func NewGCS(ctx context.Context, cfg GCSConfig) (*GCS, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("log archive GCS bucket is empty")
	}

	var (
		opts   []option.ClientOption
		source string
	)

	switch {
	case cfg.CredentialsFile != "":
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, cfg.CredentialsFile))
		source = "service account key file " + cfg.CredentialsFile
	case os.Getenv("STORAGE_EMULATOR_HOST") != "":
		source = "storage emulator at " + os.Getenv("STORAGE_EMULATOR_HOST")
	default:
		source = "application default credentials"
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating GCS client: %w", err)
	}

	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		prefix += "/"
	}

	return &GCS{client: client, bucket: client.Bucket(cfg.Bucket), prefix: prefix, credentialSource: source}, nil
}

// CredentialSource describes where the client takes its credentials from.
func (b *GCS) CredentialSource() string {
	return b.credentialSource
}

// Put uploads the object. A failed read aborts the upload, so no partial
// object is written.
func (b *GCS) Put(ctx context.Context, key string, content io.Reader) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	writer := b.bucket.Object(b.prefix + key).NewWriter(ctx)

	if _, err := io.Copy(writer, content); err != nil {
		cancel()

		return errors.Join(fmt.Errorf("uploading log archive object %s: %w", key, err), writer.Close())
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("uploading log archive object %s: %w", key, err)
	}

	return nil
}

// Open streams the object. Returns ErrNotFound when it is missing.
func (b *GCS) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	reader, err := b.bucket.Object(b.prefix + key).NewReader(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	if err != nil {
		return nil, fmt.Errorf("downloading log archive object %s: %w", key, err)
	}

	return reader, nil
}

// Delete removes the objects. Missing objects are skipped.
func (b *GCS) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := ValidateKey(key); err != nil {
			return err
		}
	}

	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(gcsDeleteConcurrency)

	for _, key := range keys {
		group.Go(func() error {
			err := b.bucket.Object(b.prefix + key).Delete(ctx)
			if err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
				return fmt.Errorf("deleting log archive object %s: %w", key, err)
			}

			return nil
		})
	}

	return group.Wait()
}

// Close releases the client's connections.
func (b *GCS) Close() error {
	return b.client.Close()
}
