package pipelineexecdocker

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
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

	cli, err := client.New(
		client.WithHost("tcp://"+strings.TrimPrefix(daemon.URL, "http://")),
		client.WithHTTPClient(daemon.Client()),
		client.WithAPIVersion("1.47"),
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

// fakeAttachDaemon serves the Docker API calls Run makes. The container exits
// as soon as it starts: /wait answers immediately and exited is closed. attach
// gets the hijacked connection and its buffered reader once the client switched
// protocols.
func fakeAttachDaemon(t *testing.T, attach func(conn net.Conn, rw *bufio.ReadWriter, exited <-chan struct{})) *client.Client {
	t.Helper()

	exited := make(chan struct{})

	var exitOnce sync.Once

	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"ctr-1","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/attach"):
			conn, rw, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)

				return
			}
			defer func() { _ = conn.Close() }()

			_, _ = rw.WriteString("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
			_ = rw.Flush()

			attach(conn, rw, exited)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/wait"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"StatusCode":0}`))
			w.(http.Flusher).Flush()
			exitOnce.Do(func() { close(exited) })
		default:
			http.Error(w, "unsupported in test", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(daemon.Close)

	cli, err := client.New(
		client.WithHost("tcp://"+strings.TrimPrefix(daemon.URL, "http://")),
		client.WithAPIVersion("1.47"),
	)
	require.NoError(t, err)

	return cli
}

// stdWriter frames each write as one multiplexed attach-stream message.
type stdWriter struct {
	w      io.Writer
	stream stdcopy.StdType
}

func newStdWriter(w io.Writer, stream stdcopy.StdType) *stdWriter {
	return &stdWriter{w: w, stream: stream}
}

func (s *stdWriter) Write(p []byte) (int, error) {
	header := make([]byte, 8)
	header[0] = byte(s.stream)
	binary.BigEndian.PutUint32(header[4:], uint32(len(p)))

	if _, err := s.w.Write(header); err != nil {
		return 0, err
	}

	return s.w.Write(p)
}

func recordLines(from, to int) string {
	var sb strings.Builder
	for i := from; i < to; i++ {
		fmt.Fprintf(&sb, "{\"type\":\"RECORD\",\"n\":%d}\n", i)
	}

	return sb.String()
}

func TestContainerRunner_Run_DrainsOutputAfterExit(t *testing.T) {
	cli := fakeAttachDaemon(t, func(_ net.Conn, rw *bufio.ReadWriter, exited <-chan struct{}) {
		stdout := newStdWriter(rw, stdcopy.Stdout)

		_, _ = stdout.Write([]byte(recordLines(0, 10)))
		_ = rw.Flush()

		<-exited

		// The rest of the output reaches the client only after the runner has
		// seen the exit and closed stdin to the container.
		_, _ = io.Copy(io.Discard, rw)

		time.Sleep(100 * time.Millisecond)

		_, _ = stdout.Write([]byte(recordLines(10, 100)))
		_ = rw.Flush()
	})

	stdinReader, stdinWriter := io.Pipe()

	t.Cleanup(func() { _ = stdinWriter.Close() })

	r := &ContainerRunner{client: cli, ownerID: "self", drainTimeout: 5 * time.Second}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1", Stdin: stdinReader})
	require.NoError(t, err)

	out, err := io.ReadAll(result.Stdout)
	require.NoError(t, err)
	assert.Equal(t, recordLines(0, 100), string(out))

	<-result.Done
	assert.Equal(t, 0, result.ExitCode)
}

func TestContainerRunner_Run_OutputThatNeverEndsFailsAfterGracePeriod(t *testing.T) {
	release := make(chan struct{})

	t.Cleanup(func() { close(release) })

	cli := fakeAttachDaemon(t, func(_ net.Conn, rw *bufio.ReadWriter, _ <-chan struct{}) {
		_, _ = newStdWriter(rw, stdcopy.Stdout).Write([]byte(recordLines(0, 3)))
		_ = rw.Flush()

		<-release
	})

	r := &ContainerRunner{client: cli, ownerID: "self", drainTimeout: 200 * time.Millisecond}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1"})
	require.NoError(t, err)

	readDone := make(chan struct{})

	var (
		out     []byte
		readErr error
	)

	go func() {
		defer close(readDone)

		out, readErr = io.ReadAll(result.Stdout)
	}()

	select {
	case <-readDone:
	case <-time.After(5 * time.Second):
		t.Fatal("reading stdout hung after the container exited")
	}

	assert.Equal(t, recordLines(0, 3), string(out))
	require.ErrorIs(t, readErr, ErrOutputDrainTimeout)

	select {
	case <-result.Done:
	case <-time.After(5 * time.Second):
		t.Fatal("Done not closed after the grace period")
	}
}

func TestContainerRunner_Run_SlowConsumerDrainsPastGracePeriod(t *testing.T) {
	cli := fakeAttachDaemon(t, func(_ net.Conn, rw *bufio.ReadWriter, exited <-chan struct{}) {
		<-exited

		stdout := newStdWriter(rw, stdcopy.Stdout)
		for i := 0; i < 1000; i += 100 {
			_, _ = stdout.Write([]byte(recordLines(i, i+100)))
		}

		_ = rw.Flush()
	})

	r := &ContainerRunner{client: cli, ownerID: "self", drainTimeout: 150 * time.Millisecond}

	result, err := r.Run(context.Background(), RunOptions{Image: "connector:1"})
	require.NoError(t, err)

	time.Sleep(400 * time.Millisecond)

	var out []byte

	buf := make([]byte, 64*1024)

	for {
		n, err := result.Stdout.Read(buf)
		out = append(out, buf[:n]...)

		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		time.Sleep(250 * time.Millisecond)
	}

	assert.Equal(t, recordLines(0, 1000), string(out))

	<-result.Done
	assert.Equal(t, 0, result.ExitCode)
}
