package logarchive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"go.uber.org/multierr"
)

// Filesystem stores objects as files under a base directory.
type Filesystem struct {
	dir string
}

// NewFilesystem returns a backend rooted at dir. The directory is created on
// the first write.
func NewFilesystem(dir string) (*Filesystem, error) {
	if dir == "" {
		return nil, errors.New("log archive directory is empty")
	}

	return &Filesystem{dir: dir}, nil
}

// Put writes the object to a temporary file and renames it into place, so
// readers never see a partial object.
func (f *Filesystem) Put(_ context.Context, key string, body io.Reader) (rerr error) {
	if err := ValidateKey(key); err != nil {
		return err
	}

	if err := os.MkdirAll(f.dir, 0o750); err != nil {
		return fmt.Errorf("creating log archive directory: %w", err)
	}

	root, err := os.OpenRoot(f.dir)
	if err != nil {
		return fmt.Errorf("opening log archive directory: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(root))

	if err := root.MkdirAll(path.Dir(key), 0o750); err != nil {
		return fmt.Errorf("creating log archive directory for %s: %w", key, err)
	}

	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return fmt.Errorf("generating temporary file name: %w", err)
	}

	tmp := key + ".tmp-" + hex.EncodeToString(suffix)

	if err := writeFile(root, tmp, body); err != nil {
		return multierr.Append(err, ignoreNotExist(root.Remove(tmp)))
	}

	if err := root.Rename(tmp, key); err != nil {
		return multierr.Append(fmt.Errorf("renaming log archive object %s: %w", key, err), ignoreNotExist(root.Remove(tmp)))
	}

	return nil
}

func writeFile(root *os.Root, name string, body io.Reader) (rerr error) {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("creating %s: %w", name, err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(file))

	if _, err := io.Copy(file, body); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("syncing %s: %w", name, err)
	}

	return nil
}

// Open returns the object's content. Returns ErrNotFound when it is missing.
func (f *Filesystem) Open(_ context.Context, key string) (_ io.ReadCloser, rerr error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(f.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	if err != nil {
		return nil, fmt.Errorf("opening log archive directory: %w", err)
	}

	defer multierr.AppendInvoke(&rerr, multierr.Close(root))

	file, err := root.Open(key)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	if err != nil {
		return nil, fmt.Errorf("opening log archive object %s: %w", key, err)
	}

	return file, nil
}

// Delete removes the objects. Missing objects are skipped.
func (f *Filesystem) Delete(_ context.Context, keys ...string) (rerr error) {
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := ValidateKey(key); err != nil {
			return err
		}
	}

	root, err := os.OpenRoot(f.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("opening log archive directory: %w", err)
	}

	defer multierr.AppendInvoke(&rerr, multierr.Close(root))

	for _, key := range keys {
		if err := ignoreNotExist(root.Remove(key)); err != nil {
			return fmt.Errorf("deleting log archive object %s: %w", key, err)
		}
	}

	return nil
}

func ignoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}
