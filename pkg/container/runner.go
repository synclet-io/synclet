package container

import (
	"context"
	"io"
)

// Runner abstracts container execution (Docker, Kubernetes, etc.).
type Runner interface {
	Run(ctx context.Context, opts RunOptions) (*RunResult, error)
	Pull(ctx context.Context, image string) error
	// ResolveDigest returns the digest-pinned image reference (e.g. "repo@sha256:abc...")
	// for a locally available image. Used to pin images by digest after pull (SEC-13).
	ResolveDigest(ctx context.Context, image string) (string, error)
	Stop(ctx context.Context, id string) error
	StopWithTimeout(ctx context.Context, id string, timeoutSeconds int) error
	Remove(ctx context.Context, id string) error
}

// RunOptions configures a container run.
type RunOptions struct {
	Image         string
	Command       []string
	ConfigFile    []byte
	CatalogFile   []byte
	StateFile     []byte
	Stdin         io.Reader
	MemoryLimit   int64
	CPULimit      float64
	MemoryRequest int64   // bytes
	CPURequest    float64 // cores
	NetworkMode   string
	Name          string
	Labels        map[string]string
	// ExtraMounts is a list of additional host-path bind mounts (Docker only, K8s ignores).
	ExtraMounts []BindMount
	// LocalDir is the host directory mounted read-write at /local (Docker
	// only, K8s ignores). Empty gives the container a directory of its own,
	// removed when the container exits.
	LocalDir string
}

// BindMount describes an extra host-path bind mount.
type BindMount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// RunResult holds handles to a running container's I/O and lifecycle.
type RunResult struct {
	Stdout      io.ReadCloser
	Stderr      io.ReadCloser
	ExitCode    int
	Done        <-chan struct{}
	ContainerID string
}
