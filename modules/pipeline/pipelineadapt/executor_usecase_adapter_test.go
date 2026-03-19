package pipelineadapt

import (
	"testing"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
)

// Compile-time check that UseCaseExecutorBackend implements ExecutorBackend.
var _ pipelinesync.ExecutorBackend = (*UseCaseExecutorBackend)(nil)

func TestUseCaseExecutorBackend_ImplementsInterface(t *testing.T) {
	// This test exists purely to verify the interface implementation at compile time.
	// If UseCaseExecutorBackend doesn't implement all ExecutorBackend methods,
	// the var _ check above will cause a compile error.
	t.Log("UseCaseExecutorBackend implements pipelinesync.ExecutorBackend")
}
