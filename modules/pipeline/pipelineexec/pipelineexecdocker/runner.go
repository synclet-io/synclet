package pipelineexecdocker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
	"go.uber.org/multierr"

	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

const defaultStderrMaxSize = 10 * 1024 * 1024 // 10MB

// defaultDrainTimeout bounds how long the attach stream may stay idle after the
// container exits before the runner gives up on reading the rest of its output.
const defaultDrainTimeout = 30 * time.Second

// ErrOutputDrainTimeout is the stdout read error of a container whose attach
// stream stayed open without data for the drain timeout after the container exited.
var ErrOutputDrainTimeout = errors.New("container exited but its output stream did not end")

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

func (b *boundedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// If the incoming data alone exceeds maxSize, keep only the tail.
	if len(data) >= b.maxSize {
		b.buf = make([]byte, b.maxSize)
		copy(b.buf, data[len(data)-b.maxSize:])

		return len(data), nil
	}
	// Discard oldest data to make room.
	if len(b.buf)+len(data) > b.maxSize {
		overflow := len(b.buf) + len(data) - b.maxSize
		b.buf = b.buf[overflow:]
	}

	b.buf = append(b.buf, data...)

	return len(data), nil
}

func (b *boundedBuffer) Read(dest []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.buf) == 0 {
		return 0, io.EOF
	}

	n := copy(dest, b.buf)
	b.buf = b.buf[n:]

	return n, nil
}

// drainingReader reads a container's attach stream. Once the container has
// exited, every read gets a deadline of timeout, so a stream that stays open
// without data ends with os.ErrDeadlineExceeded instead of blocking forever.
type drainingReader struct {
	r       io.Reader
	conn    net.Conn
	timeout time.Duration
	exited  atomic.Bool
}

func (d *drainingReader) Read(p []byte) (int, error) {
	if d.exited.Load() {
		_ = d.conn.SetReadDeadline(time.Now().Add(d.timeout))
	}

	return d.r.Read(p)
}

// startDrain bounds the read in progress, and every later one, by the timeout.
func (d *drainingReader) startDrain() {
	d.exited.Store(true)
	_ = d.conn.SetReadDeadline(time.Now().Add(d.timeout))
}

// Compile-time interface check.
var _ pkgcontainer.Runner = (*ContainerRunner)(nil)

// ContainerRunner manages Docker containers for running Airbyte connectors.
type ContainerRunner struct {
	client       *client.Client
	tempDirRoot  string
	localRoot    string
	ownerID      string
	drainTimeout time.Duration
}

// NewContainerRunner creates a new ContainerRunner using the default Docker client
// configuration (DOCKER_HOST env var, or the default Docker socket).
//
// tempDirRoot is the directory under which per-task scratch directories are
// created. Pass "" for the OS default. In Docker-in-Docker deployments this
// must point at a path bind-mounted to the same location on the host so the
// host Docker daemon can resolve the bind-mount source — see
// pipelineConfig.DockerTempDirRoot for the wiring.
//
// localRoot, when set, is the host directory mounted at /local in every
// container instead of a per-run directory under tempDirRoot; the runner never
// removes it. It follows the same host-path rule as tempDirRoot.
//
// ownerID is stamped as the OwnerLabel on every container the runner creates.
func NewContainerRunner(tempDirRoot, localRoot, ownerID string) (*ContainerRunner, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	return &ContainerRunner{client: cli, tempDirRoot: tempDirRoot, localRoot: localRoot, ownerID: ownerID}, nil
}

// OwnerID returns the owner id stamped on the runner's containers.
func (r *ContainerRunner) OwnerID() string {
	return r.ownerID
}

// containerLabels returns a copy of labels with the owner label set.
func (r *ContainerRunner) containerLabels(labels map[string]string) map[string]string {
	out := make(map[string]string, len(labels)+1)
	maps.Copy(out, labels)
	out[OwnerLabel] = r.ownerID

	return out
}

func (r *ContainerRunner) outputDrainTimeout() time.Duration {
	if r.drainTimeout > 0 {
		return r.drainTimeout
	}

	return defaultDrainTimeout
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
	reader, err := r.client.ImagePull(ctx, img, client.ImagePullOptions{})
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

		tempDir, err = CreateTempDir(r.tempDirRoot, files)
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
	}

	// A container run without a caller-owned /local directory gets the local
	// root, or else its own directory, removed together with the temp dir.
	localDir := opts.LocalDir
	releaseLocalDir := func() {}

	if localDir == "" {
		var err error

		localDir, releaseLocalDir, err = acquireLocalDir(r.tempDirRoot, r.localRoot)
		if err != nil {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}

			return nil, fmt.Errorf("creating local dir: %w", err)
		}
	}

	// Clean up helper used on error paths and once the container is done.
	cleanup := func() {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}

		releaseLocalDir()
	}

	networkMode := opts.NetworkMode
	if networkMode == "" {
		networkMode = "none"
	}

	// Container configuration.
	containerCfg := &dockercontainer.Config{
		Image:        opts.Image,
		Cmd:          opts.Command,
		Labels:       r.containerLabels(opts.Labels),
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

	hostCfg.Mounts = append(hostCfg.Mounts, mount.Mount{
		Type:     mount.TypeBind,
		Source:   localDir,
		Target:   "/local",
		ReadOnly: false,
	})

	for _, m := range opts.ExtraMounts {
		hostCfg.Mounts = append(hostCfg.Mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.HostPath,
			Target:   m.ContainerPath,
			ReadOnly: m.ReadOnly,
		})
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
	resp, err := r.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     containerCfg,
		HostConfig: hostCfg,
		Name:       opts.Name,
	})
	if err != nil {
		cleanup()

		return nil, fmt.Errorf("creating container: %w", err)
	}

	containerID := resp.ID

	// Attach to container to get stdout/stderr streams.
	attachResp, err := r.client.ContainerAttach(ctx, containerID, client.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  opts.Stdin != nil,
	})
	if err != nil {
		// Use context.WithoutCancel so cleanup succeeds even if ctx was cancelled.
		_, _ = r.client.ContainerRemove(context.WithoutCancel(ctx), containerID, client.ContainerRemoveOptions{Force: true})

		cleanup()

		return nil, fmt.Errorf("attaching to container: %w", err)
	}

	// Start container.
	if _, err := r.client.ContainerStart(ctx, containerID, client.ContainerStartOptions{}); err != nil {
		attachResp.Close()

		_, _ = r.client.ContainerRemove(context.WithoutCancel(ctx), containerID, client.ContainerRemoveOptions{Force: true})

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

	drainTimeout := r.outputDrainTimeout()
	attachOut := &drainingReader{r: attachResp.Reader, conn: attachResp.Conn, timeout: drainTimeout}

	// exitCodeReady signals that result.ExitCode has been set by the exit-monitor.
	exitCodeReady := make(chan struct{})

	// Monitor container exit via the Docker API. Once the container exits, stdin
	// is closed and the attach stream is read to its end, each read bounded by
	// the drain timeout. If waiting fails (ctx cancelled), the attach connection
	// is closed at once.
	go func() {
		wait := r.client.ContainerWait(ctx, containerID, client.ContainerWaitOptions{Condition: dockercontainer.WaitConditionNotRunning})

		waitFailed := false

		select {
		case err := <-wait.Error:
			if err != nil {
				result.ExitCode = -1
				waitFailed = true
			}
		case status := <-wait.Result:
			result.ExitCode = int(status.StatusCode)
		}

		close(exitCodeReady)

		// Close stdin to unblock the stdin copy goroutine and the router's pipe writes.
		if opts.Stdin != nil {
			if closer, ok := opts.Stdin.(io.Closer); ok {
				_ = closer.Close()
			}
		}

		if waitFailed {
			attachResp.Close()

			return
		}

		attachOut.startDrain()
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

		// StdCopy demultiplexes the Docker attach stream into stdout and stderr.
		_, copyErr := stdcopy.StdCopy(stdoutWriter, stderrBuf, attachOut)

		// Wait for exit code to be set before signaling done to callers.
		<-exitCodeReady

		attachResp.Close()

		cleanup()

		if errors.Is(copyErr, os.ErrDeadlineExceeded) {
			_ = stdoutWriter.CloseWithError(fmt.Errorf("%w: no output for %s after exit", ErrOutputDrainTimeout, drainTimeout))

			return
		}

		_ = stdoutWriter.Close()
	}()

	return result, nil
}

// ListByLabel returns all containers matching every given label filter.
func (r *ContainerRunner) ListByLabel(ctx context.Context, labels ...string) ([]dockercontainer.Summary, error) {
	list, err := r.client.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", labels...),
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	return list.Items, nil
}

// Stop stops a running container with a default timeout.
func (r *ContainerRunner) Stop(ctx context.Context, containerID string) error {
	if _, err := r.client.ContainerStop(ctx, containerID, client.ContainerStopOptions{}); err != nil {
		return fmt.Errorf("stopping container %s: %w", containerID, err)
	}

	return nil
}

// StopWithTimeout stops a container with a specific grace period.
// Docker SDK handles SIGTERM -> wait timeout -> SIGKILL automatically.
func (r *ContainerRunner) StopWithTimeout(ctx context.Context, containerID string, timeoutSeconds int) error {
	if _, err := r.client.ContainerStop(ctx, containerID, client.ContainerStopOptions{
		Timeout: &timeoutSeconds,
	}); err != nil {
		return fmt.Errorf("stopping container %s with %ds timeout: %w", containerID, timeoutSeconds, err)
	}

	return nil
}

// Remove removes a container. Use Force option if the container might still be running.
func (r *ContainerRunner) Remove(ctx context.Context, containerID string) error {
	if _, err := r.client.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("removing container %s: %w", containerID, err)
	}

	return nil
}
