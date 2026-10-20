package pipelineexecdocker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// hostConfigDaemon serves the Docker API calls Run makes, records the host
// config of the created container and lets the container exit at once.
type hostConfigDaemon struct {
	mu      sync.Mutex
	hostCfg *dockercontainer.HostConfig
}

func (d *hostConfigDaemon) created() *dockercontainer.HostConfig {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.hostCfg
}

// newHostConfigDaemon starts the daemon. With failAttach the attach call
// fails, so Run takes its error path.
func newHostConfigDaemon(t *testing.T, failAttach bool) (*hostConfigDaemon, *client.Client) {
	t.Helper()

	d := &hostConfigDaemon{}
	exited := make(chan struct{})

	var exitOnce sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			var body struct {
				HostConfig *dockercontainer.HostConfig
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			d.mu.Lock()
			d.hostCfg = body.HostConfig
			d.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"ctr-1","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/attach") && !failAttach:
			conn, rw, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)

				return
			}
			defer func() { _ = conn.Close() }()

			_, _ = rw.WriteString("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
			_ = rw.Flush()

			<-exited
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/wait"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"StatusCode":0}`))
			w.(http.Flusher).Flush()
			exitOnce.Do(func() { close(exited) })
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unsupported in test", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	cli, err := client.New(
		client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")),
		client.WithAPIVersion("1.47"),
	)
	require.NoError(t, err)

	return d, cli
}

func localMount(t *testing.T, hostCfg *dockercontainer.HostConfig) mount.Mount {
	t.Helper()

	require.NotNil(t, hostCfg, "Run did not create a container")

	for _, m := range hostCfg.Mounts {
		if m.Target == "/local" {
			return m
		}
	}

	require.Failf(t, "no /local mount", "mounts: %+v", hostCfg.Mounts)

	return mount.Mount{}
}

func TestContainerRunner_Run_MountsWritableLocalDirRemovedOnExit(t *testing.T) {
	root := t.TempDir()
	daemon, cli := newHostConfigDaemon(t, false)

	r := &ContainerRunner{client: cli, tempDirRoot: root, ownerID: "self"}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1", Command: []string{"spec"}})
	require.NoError(t, err)

	m := localMount(t, daemon.created())
	assert.Equal(t, mount.TypeBind, m.Type)
	assert.False(t, m.ReadOnly)
	assert.Equal(t, root, filepath.Dir(m.Source))

	info, err := os.Stat(m.Source)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0o777), info.Mode().Perm(), "a connector of any uid writes to /local")

	_, _ = io.Copy(io.Discard, result.Stdout)
	<-result.Done

	_, err = os.Stat(m.Source)
	assert.ErrorIs(t, err, os.ErrNotExist, "the /local directory outlived its container")
}

func TestContainerRunner_Run_RemovesLocalDirWhenStartFails(t *testing.T) {
	root := t.TempDir()
	daemon, cli := newHostConfigDaemon(t, true)

	r := &ContainerRunner{client: cli, tempDirRoot: root, ownerID: "self"}

	_, err := r.Run(context.Background(), RunOptions{Image: "connector:1"})
	require.Error(t, err)

	m := localMount(t, daemon.created())

	_, err = os.Stat(m.Source)
	assert.ErrorIs(t, err, os.ErrNotExist, "the /local directory outlived a container that failed to start")
}

func TestContainerRunner_Run_MountsGivenLocalDirAndKeepsIt(t *testing.T) {
	jobDir := t.TempDir()
	daemon, cli := newHostConfigDaemon(t, false)

	r := &ContainerRunner{client: cli, tempDirRoot: t.TempDir(), ownerID: "self"}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1", LocalDir: jobDir})
	require.NoError(t, err)

	m := localMount(t, daemon.created())
	assert.Equal(t, jobDir, m.Source)
	assert.False(t, m.ReadOnly)

	_, _ = io.Copy(io.Discard, result.Stdout)
	<-result.Done

	_, err = os.Stat(jobDir)
	assert.NoError(t, err, "the runner removed a directory its caller owns")
}

// localDirRunner records the /local directory of every run and whether it
// existed when the container started. Runs past failFrom fail.
type localDirRunner struct {
	mockContainerRunner

	mu       sync.Mutex
	dirs     []string
	existed  []bool
	failFrom int
}

func (r *localDirRunner) Run(_ context.Context, opts pkgcontainer.RunOptions) (*pkgcontainer.RunResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, statErr := os.Stat(opts.LocalDir)
	r.dirs = append(r.dirs, opts.LocalDir)
	r.existed = append(r.existed, opts.LocalDir != "" && statErr == nil)

	if len(r.dirs) > r.failFrom {
		return nil, errors.New("container start failed")
	}

	done := make(chan struct{})
	close(done)

	return &pkgcontainer.RunResult{
		Stdout: io.NopCloser(strings.NewReader("")),
		Stderr: io.NopCloser(strings.NewReader("")),
		Done:   done,
	}, nil
}

type discardLog struct{}

func (discardLog) Info(context.Context, string, ...interface{})         {}
func (discardLog) Warn(context.Context, string, ...interface{})         {}
func (discardLog) Debug(context.Context, string, ...interface{})        {}
func (discardLog) Error(context.Context, string, ...interface{})        {}
func (d discardLog) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d discardLog) WithField(string, interface{}) logging.Delegate     { return d }
func (d discardLog) WithError(error) logging.Delegate                   { return d }
func (d discardLog) Named(string) logging.Delegate                      { return d }
func (d discardLog) SkipCallers(int) logging.Delegate                   { return d }

func TestDockerSyncExecutor_SharesOneLocalDirPerJobAndRemovesIt(t *testing.T) {
	root := t.TempDir()
	runner := &localDirRunner{failFrom: 1}

	executor := NewDockerSyncExecutor(DockerSyncExecutorParams{
		Client:      NewConnectorClient(runner),
		TempDirRoot: root,
		Logger:      &logging.Logger{Delegate: discardLog{}},
	})

	_, err := executor.Execute(context.Background(), &pipelineexec.SyncBundle{
		Job:               &pipelineservice.Job{ID: uuid.New()},
		SourceImage:       "source:1",
		DestImage:         "destination:1",
		SourceConfig:      json.RawMessage(`{}`),
		DestConfig:        json.RawMessage(`{}`),
		ConfiguredCatalog: &protocol.ConfiguredAirbyteCatalog{},
	}, func() {})
	require.Error(t, err)

	require.Len(t, runner.dirs, 2, "source and destination containers")
	assert.NotEmpty(t, runner.dirs[0])
	assert.Equal(t, runner.dirs[0], runner.dirs[1], "source and destination share the job's /local")
	assert.Equal(t, root, filepath.Dir(runner.dirs[0]))
	assert.Equal(t, []bool{true, true}, runner.existed)

	_, err = os.Stat(runner.dirs[0])
	assert.ErrorIs(t, err, os.ErrNotExist, "the job's /local directory outlived the job")
}
