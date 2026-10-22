package pipelineexecdocker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// localRootWithFile returns a local root holding one file a previous job wrote.
func localRootWithFile(t *testing.T) (root, file string) {
	t.Helper()

	root = t.TempDir()
	file = filepath.Join(root, "previous.duckdb")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0o600))

	return root, file
}

func assertLocalRootKept(t *testing.T, root, file string) {
	t.Helper()

	info, err := os.Stat(root)
	require.NoError(t, err, "the local root was removed")
	assert.True(t, info.IsDir())

	_, err = os.Stat(file)
	assert.NoError(t, err, "a file in the local root was removed")
}

func assertNoJobLocalDirs(t *testing.T, tempDirRoot string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(tempDirRoot, "synclet-local-*"))
	require.NoError(t, err)
	assert.Empty(t, matches, "a per-job /local directory was created although the local root is set")
}

func TestContainerRunner_Run_MountsLocalRootAndKeepsIt(t *testing.T) {
	root, file := localRootWithFile(t)
	tempDirRoot := t.TempDir()
	daemon, cli := newHostConfigDaemon(t, false)

	r := &ContainerRunner{client: cli, tempDirRoot: tempDirRoot, localRoot: root, ownerID: "self"}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1", Command: []string{"check"}})
	require.NoError(t, err)

	m := localMount(t, daemon.created())
	assert.Equal(t, root, m.Source)
	assert.False(t, m.ReadOnly)
	assertNoJobLocalDirs(t, tempDirRoot)

	_, _ = io.Copy(io.Discard, result.Stdout)
	<-result.Done

	assertLocalRootKept(t, root, file)
}

func TestContainerRunner_Run_KeepsLocalRootWhenStartFails(t *testing.T) {
	root, file := localRootWithFile(t)
	daemon, cli := newHostConfigDaemon(t, true)

	r := &ContainerRunner{client: cli, tempDirRoot: t.TempDir(), localRoot: root, ownerID: "self"}

	_, err := r.Run(context.Background(), RunOptions{Image: "connector:1"})
	require.Error(t, err)

	assert.Equal(t, root, localMount(t, daemon.created()).Source)
	assertLocalRootKept(t, root, file)
}

func TestContainerRunner_Run_CreatesMissingLocalRootWritableByAnyUID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "airbyte_local")
	daemon, cli := newHostConfigDaemon(t, false)

	r := &ContainerRunner{client: cli, tempDirRoot: t.TempDir(), localRoot: root, ownerID: "self"}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1"})
	require.NoError(t, err)

	assert.Equal(t, root, localMount(t, daemon.created()).Source)

	_, _ = io.Copy(io.Discard, result.Stdout)
	<-result.Done

	assertStickyWorldWritable(t, root)
}

func assertStickyWorldWritable(t *testing.T, root string) {
	t.Helper()

	info, err := os.Stat(root)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o777), info.Mode().Perm(), "a connector of any uid writes to /local")
	assert.NotZero(t, info.Mode()&os.ModeSticky, "a created root is sticky, so users cannot remove each other's files")
}

func TestEnsureLocalRoot_CreatesMissingRootSticky(t *testing.T) {
	root := filepath.Join(t.TempDir(), "airbyte_local")

	require.NoError(t, EnsureLocalRoot(root))

	assertStickyWorldWritable(t, root)
}

func TestEnsureLocalRoot_LeavesExistingRootAsIs(t *testing.T) {
	root, file := localRootWithFile(t)
	require.NoError(t, os.Chmod(root, 0o750)) //nolint:gosec // G302: a root the operator made, not writable by others.

	require.NoError(t, EnsureLocalRoot(root))

	info, err := os.Stat(root)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o750), info.Mode().Perm(), "the operator's permissions were changed")
	assert.Zero(t, info.Mode()&os.ModeSticky)
	assertLocalRootKept(t, root, file)
}

// warnLog records the warnings logged through it, with their fields.
type warnLog struct {
	discardLog

	fields   map[string]interface{}
	warnings *[]string
}

func (l warnLog) Warn(_ context.Context, msg string, _ ...interface{}) {
	*l.warnings = append(*l.warnings, fmt.Sprintf("%s %v", msg, l.fields))
}

func (l warnLog) WithFields(fields map[string]interface{}) logging.Delegate {
	merged := map[string]interface{}{}
	maps.Copy(merged, l.fields)
	maps.Copy(merged, fields)
	l.fields = merged

	return l
}

func (l warnLog) WithField(key string, value interface{}) logging.Delegate {
	return l.WithFields(map[string]interface{}{key: value})
}

func (l warnLog) WithError(error) logging.Delegate { return l }
func (l warnLog) Named(string) logging.Delegate    { return l }
func (l warnLog) SkipCallers(int) logging.Delegate { return l }

func TestPrepareLocalRoot_WarnsWhenRootIsNotWritableByOthers(t *testing.T) {
	var warnings []string

	logger := &logging.Logger{Delegate: warnLog{warnings: &warnings}}

	root, _ := localRootWithFile(t)
	require.NoError(t, os.Chmod(root, 0o750)) //nolint:gosec // G302: a root the operator made, not writable by others.

	require.NoError(t, PrepareLocalRoot(context.Background(), root, logger))
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], root)
	assert.Contains(t, warnings[0], "drwxr-x---")

	warnings = nil

	require.NoError(t, PrepareLocalRoot(context.Background(), filepath.Join(t.TempDir(), "created"), logger))
	assert.Empty(t, warnings, "a root Synclet creates is writable by any uid")
}

func TestDockerSyncExecutor_MountsLocalRootAndKeepsIt(t *testing.T) {
	root, file := localRootWithFile(t)
	tempDirRoot := t.TempDir()
	runner := &localDirRunner{failFrom: 2}

	executor := NewDockerSyncExecutor(DockerSyncExecutorParams{
		Client:      NewConnectorClient(runner),
		TempDirRoot: tempDirRoot,
		LocalRoot:   root,
		Logger:      &logging.Logger{Delegate: discardLog{}},
	})

	_, _ = executor.Execute(context.Background(), &pipelineexec.SyncBundle{
		Job:               &pipelineservice.Job{ID: uuid.New()},
		SourceImage:       "source:1",
		DestImage:         "destination:1",
		SourceConfig:      json.RawMessage(`{}`),
		DestConfig:        json.RawMessage(`{}`),
		ConfiguredCatalog: &protocol.ConfiguredAirbyteCatalog{},
	}, func() {})

	require.Len(t, runner.dirs, 2, "source and destination containers")
	assert.Equal(t, []string{root, root}, runner.dirs)
	assertNoJobLocalDirs(t, tempDirRoot)
	assertLocalRootKept(t, root, file)
}

func TestContainerRunner_Run_MountsOnlyScratchAndLocalRoot(t *testing.T) {
	const runtimeConfig = `{"docker_volumes":[{"host_path":"/var/run/docker.sock","container_path":"/var/run/docker.sock"}]}`

	catalog := &protocol.ConfiguredAirbyteCatalog{}
	config := json.RawMessage(`{}`)
	redactor := pipelineexec.NewRedactor(config)

	operations := map[string]func(c *ConnectorClient, limits ResourceLimits){
		"check": func(c *ConnectorClient, limits ResourceLimits) {
			_, _ = c.Check(context.Background(), "connector:1", config, limits)
		},
		"read": func(c *ConnectorClient, limits ResourceLimits) {
			out, cleanup, err := c.Read(context.Background(), "connector:1", config, redactor, catalog, nil, nil, limits)
			require.NoError(t, err)

			_, _ = io.Copy(io.Discard, out)

			_ = out.Close()

			cleanup()
		},
		"write": func(c *ConnectorClient, limits ResourceLimits) {
			out, cleanup, err := c.Write(context.Background(), "connector:1", config, redactor, catalog, strings.NewReader(""), nil, limits)
			require.NoError(t, err)

			_, _ = io.Copy(io.Discard, out)

			_ = out.Close()

			cleanup()
		},
	}

	for opName, run := range operations {
		for _, rootSet := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s local root set %v", opName, rootSet), func(t *testing.T) {
				localRoot := ""
				if rootSet {
					localRoot = t.TempDir()
				}

				tempDirRoot := t.TempDir()
				daemon, cli := newHostConfigDaemon(t, false)

				runner := &ContainerRunner{client: cli, tempDirRoot: tempDirRoot, localRoot: localRoot, ownerID: "self"}
				run(NewConnectorClient(runner), resolveResources(pipelineservice.RuntimeDefaults{}, runtimeConfig))

				hostCfg := daemon.created()
				require.NotNil(t, hostCfg, "no container was created")

				for _, m := range hostCfg.Mounts {
					assert.Contains(t, []string{"/tmp", "/local"}, m.Target, "unexpected mount %+v", m)
					assert.NotEqual(t, "/var/run/docker.sock", m.Source, "a host path from runtime_config was mounted")
				}

				local := localMount(t, hostCfg)
				assert.False(t, local.ReadOnly)

				if rootSet {
					assert.Equal(t, localRoot, local.Source)
				} else {
					assert.True(t, strings.HasPrefix(local.Source, tempDirRoot+string(filepath.Separator)), "per-job /local %q is not under %q", local.Source, tempDirRoot)
				}
			})
		}
	}
}
