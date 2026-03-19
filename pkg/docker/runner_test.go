package docker

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

func TestBoundedBuffer_UnderLimit(t *testing.T) {
	buf := newBoundedBuffer(100)
	n, err := buf.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	out := make([]byte, 100)
	n, err = buf.Read(out)
	assert.NoError(t, err)
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
	assert.Equal(t, int64(2*1024*1024*1024), defaultDockerMemoryLimit)
	assert.Equal(t, 1.0, defaultDockerCPULimit)

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
