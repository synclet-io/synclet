package pipelineadapt

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

// TestClaimJobResultMirrorsBundleResult fails when pipelineexec.ClaimJobResult
// and pipelinejobs.ClaimJobBundleResult differ in a field, so that a field
// added to the use-case result is also copied by UseCaseExecutorBackend.ClaimJob.
func TestClaimJobResultMirrorsBundleResult(t *testing.T) {
	assert.Equal(t, fieldSignatures(reflect.TypeOf(pipelinejobs.ClaimJobBundleResult{})), fieldSignatures(reflect.TypeOf(pipelineexec.ClaimJobResult{})))
}

func fieldSignatures(typ reflect.Type) []string {
	signatures := make([]string, 0, typ.NumField())
	for i := range typ.NumField() {
		field := typ.Field(i)
		signatures = append(signatures, field.Name+" "+field.Type.String())
	}

	return signatures
}
