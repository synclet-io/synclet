// Package logarchive stores job log archives in a filesystem directory or an
// S3-compatible bucket, or nowhere when archiving is disabled.
package logarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	// ErrDisabled is returned by None for reads and writes.
	ErrDisabled = errors.New("log archive is disabled")
	// ErrNotFound is returned by Open when no object exists under the key.
	ErrNotFound = errors.New("log archive object not found")
	// ErrInvalidKey is returned for keys that are empty, absolute or contain
	// empty, "." or ".." segments or backslashes.
	ErrInvalidKey = errors.New("invalid log archive key")
)

// None is the backend used when archiving is disabled: nothing is stored and
// deletes succeed.
type None struct{}

func (None) Put(context.Context, string, io.Reader) error { return ErrDisabled }

func (None) Open(context.Context, string) (io.ReadCloser, error) { return nil, ErrDisabled }

func (None) Delete(context.Context, ...string) error { return nil }

// ValidateKey checks that key is a relative slash-separated path that cannot
// leave the archive root.
func ValidateKey(key string) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") {
		return fmt.Errorf("%w: %q", ErrInvalidKey, key)
	}

	for segment := range strings.SplitSeq(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: %q", ErrInvalidKey, key)
		}
	}

	return nil
}
