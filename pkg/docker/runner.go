package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"go.uber.org/multierr"

	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

const defaultStderrMaxSize = 10 * 1024 * 1024 // 10MB

const (
	// defaultDockerMemoryLimit is the default memory limit for connector containers (2Gi).
	defaultDockerMemoryLimit int64 = 2 * 1024 * 1024 * 1024
	// defaultDockerCPULimit is the default CPU limit in cores (1.0).
	defaultDockerCPULimit float64 = 1.0
)

// boundedBuffer is a thread-safe buffer with a maximum size that discards
// oldest data when the limit is exceeded. Used for stderr to prevent unbounded
// memory growth from verbose connectors while keeping the most recent output.
type boundedBuffer struct {
	mu      sync.Mutex
	buf     []byte
	maxSize int
}

func newBoundedBuffer(maxSize int) *boundedBuffer {
	return &boundedBuffer{maxSize: maxSize}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// If the incoming data alone exceeds maxSize, keep only the tail.
	if len(p) >= b.maxSize {
		b.buf = make([]byte, b.maxSize)
		copy(b.buf, p[len(p)-b.maxSize:])
		return len(p), nil
	}
	// Discard oldest data to make room.
	if len(b.buf)+len(p) > b.maxSize {
		overflow := len(b.buf) + len(p) - b.maxSize
		b.buf = b.buf[overflow:]
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *boundedBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.buf)
	b.buf = b.buf[n:]
	return n, nil
}

// Compile-time interface check.
var _ pkgcontainer.Runner = (*ContainerRunner)(nil)

// ContainerRunner manages Docker containers for running Airbyte connectors.
type ContainerRunner struct {
	client *client.Client
}

// NewContainerRunner creates a new ContainerRunner using the default Docker client
// configuration (DOCKER_HOST env var, or the default Docker socket).
func NewContainerRunner() (*ContainerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return &ContainerRunner{client: cli}, nil
}

// ResolveDigest returns the digest-pinned reference for a locally available image.
// It inspects the image metadata and extracts the first sha256 RepoDigest (SEC-13).
func (r *ContainerRunner) ResolveDigest(ctx context.Context, img string) (string, error) {
	inspect, err := r.client.ImageInspect(ctx, img)
	if err != nil {
		return "", fmt.Errorf("inspecting image %s: %w", img, err)
	}
	for _, digest := range inspect.RepoDigests {
		if strings.Contains(digest, "@sha256:") {
			return digest, nil
		}
	}
	return "", fmt.Errorf("no sha256 digest found for image %s", img)
}

// Pull pulls a Docker image, discarding progress output.
func (r *ContainerRunner) Pull(ctx context.Context, img string) (rerr error) {
	reader, err := r.client.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", img, err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(reader))

	// Drain the pull progress stream; we discard it for now.
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("reading pull progress for %s: %w", img, err)
	}
	return nil
}

// Run creates and starts a container with the given options. It returns immediately
// with I/O streams and a Done channel. The caller is responsible for reading stdout/stderr
// and calling Stop/Remove when finished.
func (r *ContainerRunner) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	// Build temp dir with config/catalog/state files.
	files := make(map[string][]byte)
	if opts.ConfigFile != nil {
		files["config.json"] = opts.ConfigFile
	}
	if opts.CatalogFile != nil {
		files["catalog.json"] = opts.CatalogFile
	}
	if opts.StateFile != nil {
		files["state.json"] = opts.StateFile
	}

	var tempDir string
	if len(files) > 0 {
		var err error
		tempDir, err = CreateTempDir(files)
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
	}

	// Clean up helper used on error paths.
	cleanup := func() {
		if tempDir != "" {
			_ = CleanupTempDir(tempDir)
		}
	}

	networkMode := opts.NetworkMode
	if networkMode == "" {
		networkMode = "none"
	}

	// Container configuration.
	containerCfg := &dockercontainer.Config{
		Image:        opts.Image,
		Cmd:          opts.Command,
		Labels:       opts.Labels,
		OpenStdin:    opts.Stdin != nil,
		StdinOnce:    opts.Stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  opts.Stdin != nil,
	}

	// Host configuration with resource limits and mounts.
	hostCfg := &dockercontainer.HostConfig{
		NetworkMode: dockercontainer.NetworkMode(networkMode),
	}

	if tempDir != "" {
		hostCfg.Mounts = []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   tempDir,
				Target:   "/tmp",
				ReadOnly: false,
			},
		}
	}

	if opts.MemoryLimit > 0 {
		hostCfg.Memory = opts.MemoryLimit
	} else {
		hostCfg.Memory = defaultDockerMemoryLimit
	}
	if opts.CPULimit > 0 {
		// Docker expects NanoCPUs: 1 CPU core = 1e9 NanoCPUs.
		hostCfg.NanoCPUs = int64(opts.CPULimit * 1e9)
	} else {
		hostCfg.NanoCPUs = int64(defaultDockerCPULimit * 1e9)
	}

	// Create container.
	resp, err := r.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, opts.Name)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("creating container: %w", err)
	}

	containerID := resp.ID

	// Attach to container to get stdout/stderr streams.
	attachResp, err := r.client.ContainerAttach(ctx, containerID, dockercontainer.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  opts.Stdin != nil,
	})
	if err != nil {
		// Use context.WithoutCancel so cleanup succeeds even if ctx was cancelled.
		_ = r.client.ContainerRemove(context.WithoutCancel(ctx), containerID, dockercontainer.RemoveOptions{Force: true})
		cleanup()
		return nil, fmt.Errorf("attaching to container: %w", err)
	}

	// Start container.
	if err := r.client.ContainerStart(ctx, containerID, dockercontainer.StartOptions{}); err != nil {
		attachResp.Close()
		_ = r.client.ContainerRemove(context.WithoutCancel(ctx), containerID, dockercontainer.RemoveOptions{Force: true})
		cleanup()
		return nil, fmt.Errorf("starting container: %w", err)
	}

	// Split multiplexed Docker stream into stdout (pipe) and stderr (buffer).
	stdoutReader, stdoutWriter := io.Pipe()
	// Stderr uses a buffered writer instead of io.Pipe to prevent deadlocks.
	// io.Pipe is synchronous — Write blocks until Read consumes the data.
	// Since nobody actively reads stderr during streaming (it's only read after
	// the container exits), StdCopy would block on stderr writes whenever the
	// container outputs to stderr, deadlocking the entire pipeline.
	stderrBuf := newBoundedBuffer(defaultStderrMaxSize)

	result := &RunResult{
		Stdout:      stdoutReader,
		Stderr:      io.NopCloser(stderrBuf),
		ContainerID: containerID,
	}

	done := make(chan struct{})
	result.Done = done

	// exitCodeReady signals that result.ExitCode has been set by the exit-monitor.
	exitCodeReady := make(chan struct{})

	// Monitor container exit independently via Docker API. When the container
	// exits, close opts.Stdin and the attach connection to unblock all goroutines.
	go func() {
		statusCh, errCh := r.client.ContainerWait(ctx, containerID, dockercontainer.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				result.ExitCode = -1
			}
		case status := <-statusCh:
			result.ExitCode = int(status.StatusCode)
		}
		close(exitCodeReady)

		// Close stdin to unblock the stdin copy goroutine and the router's pipe writes.
		if opts.Stdin != nil {
			if closer, ok := opts.Stdin.(io.Closer); ok {
				_ = closer.Close()
			}
		}

		// Close the attach connection to unblock StdCopy.
		attachResp.Close()
	}()

	// Pipe stdin if provided.
	if opts.Stdin != nil {
		go func() {
			defer func() { _ = attachResp.CloseWrite() }()
			_, _ = io.Copy(attachResp.Conn, opts.Stdin)
		}()
	}

	// Background goroutine: demux streams, wait for exit code, then clean up.
	go func() {
		defer close(done)
		defer func() { _ = stdoutWriter.Close() }()

		// StdCopy demultiplexes the Docker attach stream into stdout and stderr.
		_, _ = stdcopy.StdCopy(stdoutWriter, stderrBuf, attachResp.Reader)

		// Wait for exit code to be set before signaling done to callers.
		<-exitCodeReady

		// Clean up temp directory after container exits.
		if tempDir != "" {
			_ = CleanupTempDir(tempDir)
		}
	}()

	return result, nil
}

// ListByLabel returns all containers matching the given label.
func (r *ContainerRunner) ListByLabel(ctx context.Context, label string) ([]dockercontainer.Summary, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", label)
	return r.client.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
}

// Stop stops a running container with a default timeout.
func (r *ContainerRunner) Stop(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStop(ctx, containerID, dockercontainer.StopOptions{}); err != nil {
		return fmt.Errorf("stopping container %s: %w", containerID, err)
	}
	return nil
}

// StopWithTimeout stops a container with a specific grace period.
// Docker SDK handles SIGTERM -> wait timeout -> SIGKILL automatically.
func (r *ContainerRunner) StopWithTimeout(ctx context.Context, containerID string, timeoutSeconds int) error {
	if err := r.client.ContainerStop(ctx, containerID, dockercontainer.StopOptions{
		Timeout: &timeoutSeconds,
	}); err != nil {
		return fmt.Errorf("stopping container %s with %ds timeout: %w", containerID, timeoutSeconds, err)
	}
	return nil
}

// Remove removes a container. Use Force option if the container might still be running.
func (r *ContainerRunner) Remove(ctx context.Context, containerID string) error {
	if err := r.client.ContainerRemove(ctx, containerID, dockercontainer.RemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("removing container %s: %w", containerID, err)
	}
	return nil
}
