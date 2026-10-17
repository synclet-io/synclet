package pipelineexecdocker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

func TestBoundedBuffer_UnderLimit(t *testing.T) {
	buf := newBoundedBuffer(100)
	n, err := buf.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	out := make([]byte, 100)
	n, err = buf.Read(out)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(out[:n]))
}

func TestBoundedBuffer_ExceedsLimit_DiscardsOldest(t *testing.T) {
	buf := newBoundedBuffer(10)
	_, _ = buf.Write([]byte("12345"))    // buf: "12345" (5 bytes)
	_, _ = buf.Write([]byte("67890abc")) // 5+8=13 > 10, overflow=3, discard oldest 3
	out := make([]byte, 20)
	n, _ := buf.Read(out)
	assert.Equal(t, "4567890abc", string(out[:n])) // kept most recent 10 bytes
}

func TestBoundedBuffer_SingleWriteExceedsLimit(t *testing.T) {
	buf := newBoundedBuffer(5)
	_, _ = buf.Write([]byte("1234567890")) // 10 > 5, overflow=10 >= 0, clear then keep last 5
	out := make([]byte, 10)
	n, _ := buf.Read(out)
	assert.Equal(t, "67890", string(out[:n])) // kept most recent 5
}

func TestBoundedBuffer_MultipleSmallWrites(t *testing.T) {
	buf := newBoundedBuffer(8)
	_, _ = buf.Write([]byte("abc")) // 3 bytes
	_, _ = buf.Write([]byte("def")) // 6 bytes
	_, _ = buf.Write([]byte("ghi")) // 9 > 8, overflow=1, discard oldest 1
	_, _ = buf.Write([]byte("jk"))  // 10 > 8, overflow=2, discard oldest 2
	out := make([]byte, 20)
	n, _ := buf.Read(out)
	assert.Equal(t, "defghijk", string(out[:n]))
}

func TestBoundedBuffer_EmptyRead(t *testing.T) {
	buf := newBoundedBuffer(10)
	out := make([]byte, 10)
	_, err := buf.Read(out)
	assert.ErrorIs(t, err, io.EOF)
}

func TestDefaultDockerResourceLimits(t *testing.T) {
	// Test default constant values.
	assert.Equal(t, defaultDockerMemoryLimit, int64(2*1024*1024*1024))
	assert.InDelta(t, 1.0, defaultDockerCPULimit, 0.001)

	// Test defaulting logic: when opts are 0, defaults apply.
	t.Run("zero opts get defaults", func(t *testing.T) {
		var memory int64
		var nanoCPUs int64

		opts := pkgcontainer.RunOptions{MemoryLimit: 0, CPULimit: 0}
		if opts.MemoryLimit > 0 {
			memory = opts.MemoryLimit
		} else {
			memory = defaultDockerMemoryLimit
		}

		if opts.CPULimit > 0 {
			nanoCPUs = int64(opts.CPULimit * 1e9)
		} else {
			nanoCPUs = int64(defaultDockerCPULimit * 1e9)
		}

		assert.Equal(t, int64(2*1024*1024*1024), memory)
		assert.Equal(t, int64(1e9), nanoCPUs)
	})

	// Test explicit values override defaults.
	t.Run("explicit opts override defaults", func(t *testing.T) {
		var memory int64
		var nanoCPUs int64

		opts := pkgcontainer.RunOptions{MemoryLimit: 4 * 1024 * 1024 * 1024, CPULimit: 2.0}
		if opts.MemoryLimit > 0 {
			memory = opts.MemoryLimit
		} else {
			memory = defaultDockerMemoryLimit
		}

		if opts.CPULimit > 0 {
			nanoCPUs = int64(opts.CPULimit * 1e9)
		} else {
			nanoCPUs = int64(defaultDockerCPULimit * 1e9)
		}

		assert.Equal(t, int64(4*1024*1024*1024), memory)
		assert.Equal(t, int64(2e9), nanoCPUs)
	})
}

func TestContainerRunner_ContainerLabels(t *testing.T) {
	r := &ContainerRunner{ownerID: "self"}

	assert.Equal(t, map[string]string{OwnerLabel: "self"}, r.containerLabels(nil))

	input := map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-1"}
	assert.Equal(t, map[string]string{
		"synclet.io/managed": "true",
		"synclet.io/job-id":  "job-1",
		OwnerLabel:           "self",
	}, r.containerLabels(input))
	assert.Equal(t, map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-1"}, input)
}

func TestContainerRunner_Run_StampsOwnerLabel(t *testing.T) {
	var (
		mu      sync.Mutex
		created *dockercontainer.Config
	)

	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			var cfg dockercontainer.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			mu.Lock()
			created = &cfg
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"ctr-1","Warnings":[]}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unsupported in test", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(daemon.Close)

	cli, err := client.NewClientWithOpts(
		client.WithHost("tcp://"+strings.TrimPrefix(daemon.URL, "http://")),
		client.WithHTTPClient(daemon.Client()),
		client.WithVersion("1.47"),
	)
	require.NoError(t, err)

	r := &ContainerRunner{client: cli, ownerID: "self"}

	_, err = r.Run(context.Background(), RunOptions{
		Image:  "connector:1",
		Labels: map[string]string{"synclet.io/managed": "true"},
	})
	require.Error(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.NotNil(t, created, "Run did not create a container")
	assert.Equal(t, "self", created.Labels[OwnerLabel])
	assert.Equal(t, "true", created.Labels["synclet.io/managed"])
}
