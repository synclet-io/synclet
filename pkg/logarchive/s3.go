package logarchive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/sync/errgroup"
)

const s3DeleteConcurrency = 8

// ErrNoRegion is returned by NewS3 when neither the config nor the AWS
// environment names a region.
var ErrNoRegion = errors.New("log archive S3 region is not set")

// S3Config configures the S3 backend. Endpoint is empty for AWS S3 and set for
// other S3-compatible stores. Without static keys the default AWS credential
// chain is used.
type S3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	Prefix          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

// S3 stores objects in an S3-compatible bucket.
type S3 struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3 builds the S3 client. It does not contact the store.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("log archive bucket is empty")
	}

	if (cfg.AccessKeyID == "") != (cfg.SecretAccessKey == "") {
		return nil, errors.New("log archive S3 access key and secret key must be set together")
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	}

	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}

	if cfg.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	if awsCfg.Region == "" {
		return nil, ErrNoRegion
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}

		o.UsePathStyle = cfg.UsePathStyle
	})

	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		prefix += "/"
	}

	return &S3{client: client, bucket: cfg.Bucket, prefix: prefix}, nil
}

// Put uploads the object. Readers that cannot seek are buffered in memory so
// the request can be signed and retried.
func (b *S3) Put(ctx context.Context, key string, content io.Reader) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	body, ok := content.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(content)
		if err != nil {
			return fmt.Errorf("reading log archive object %s: %w", key, err)
		}

		body = bytes.NewReader(data)
	}

	if _, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(b.prefix + key),
		Body:   body,
	}); err != nil {
		return fmt.Errorf("uploading log archive object %s: %w", key, err)
	}

	return nil
}

// Open streams the object. Returns ErrNotFound when it is missing.
func (b *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(b.prefix + key),
	})
	if err != nil {
		var noSuchKey *s3types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
		}

		return nil, fmt.Errorf("downloading log archive object %s: %w", key, err)
	}

	return out.Body, nil
}

// Delete removes the objects one request each, since multi-object delete is
// not available on every S3-compatible store. Deleting a missing object
// succeeds in S3.
func (b *S3) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := ValidateKey(key); err != nil {
			return err
		}
	}

	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(s3DeleteConcurrency)

	for _, key := range keys {
		group.Go(func() error {
			if _, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(b.bucket),
				Key:    aws.String(b.prefix + key),
			}); err != nil {
				return fmt.Errorf("deleting log archive object %s: %w", key, err)
			}

			return nil
		})
	}

	return group.Wait()
}
