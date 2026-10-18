package pipelineexec

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type heartbeatBackend struct {
	ExecutorBackend
	sent []ExecutorInfo
	err  error
}

func (b *heartbeatBackend) ExecutorHeartbeat(_ context.Context, info ExecutorInfo) error {
	b.sent = append(b.sent, info)

	return b.err
}

// errorLog records the messages logged at error level.
type errorLog struct {
	messages *[]string
}

func (l errorLog) Info(context.Context, string, ...interface{})  {}
func (l errorLog) Warn(context.Context, string, ...interface{})  {}
func (l errorLog) Debug(context.Context, string, ...interface{}) {}
func (l errorLog) Error(_ context.Context, msg string, _ ...interface{}) {
	*l.messages = append(*l.messages, msg)
}
func (l errorLog) WithFields(map[string]interface{}) logging.Delegate { return l }
func (l errorLog) WithField(string, interface{}) logging.Delegate     { return l }
func (l errorLog) WithError(error) logging.Delegate                   { return l }
func (l errorLog) Named(string) logging.Delegate                      { return l }
func (l errorLog) SkipCallers(int) logging.Delegate                   { return l }

func TestRegistrar_Execute(t *testing.T) {
	info := ExecutorInfo{ID: "docker-0", Type: pipelineservice.ExecutorTypeDocker, Version: "v0.5.0", Capacity: 5}

	t.Run("sends the executor info", func(t *testing.T) {
		backend := &heartbeatBackend{}
		var logged []string

		require.NoError(t, NewRegistrar(backend, info, &logging.Logger{Delegate: errorLog{&logged}}).Execute(t.Context()))
		assert.Equal(t, []ExecutorInfo{info}, backend.sent)
		assert.Empty(t, logged)
	})

	t.Run("a failed heartbeat is logged, not returned", func(t *testing.T) {
		backend := &heartbeatBackend{err: errors.New("control plane down")}
		var logged []string

		require.NoError(t, NewRegistrar(backend, info, &logging.Logger{Delegate: errorLog{&logged}}).Execute(t.Context()))
		assert.Len(t, backend.sent, 1)
		assert.Len(t, logged, 1)
	})
}
