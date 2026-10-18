package app

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
)

// executorGraphs are the fx graphs of the executor commands.
func executorGraphs() map[string]fx.Option {
	return map[string]fx.Option{
		"docker-executor": dockerExecutorGraph().options(),
		"k8s-executor":    k8sExecutorGraph().options(),
	}
}

// forbiddenExecutorTypes must not resolve in an executor graph: the database
// connection, storage, the use cases over it, the outbox and the event bus.
var forbiddenExecutorTypes = map[string]reflect.Type{
	"*gorm.DB":                         reflect.TypeOf((*gorm.DB)(nil)),
	"*sql.DB":                          reflect.TypeOf((*sql.DB)(nil)),
	"pipelineservice.Storage":          reflect.TypeOf((*pipelineservice.Storage)(nil)).Elem(),
	"*pipelinestorage.Storages":        reflect.TypeOf((*pipelinestorage.Storages)(nil)),
	"*pipelinejobs.ClaimJobBundle":     reflect.TypeOf((*pipelinejobs.ClaimJobBundle)(nil)),
	"*pipelinejobs.SetK8sJobName":      reflect.TypeOf((*pipelinejobs.SetK8sJobName)(nil)),
	"*pipelinestate.SaveSyncState":     reflect.TypeOf((*pipelinestate.SaveSyncState)(nil)),
	"*pipelinelogs.BatchAppendJobLogs": reflect.TypeOf((*pipelinelogs.BatchAppendJobLogs)(nil)),
	"txoutbox.MessageSender":           reflect.TypeOf((*txoutbox.MessageSender)(nil)).Elem(),
	"message.Publisher":                reflect.TypeOf((*message.Publisher)(nil)).Elem(),
}

// requestType is an fx.Invoke function that asks the graph for one value of typ.
func requestType(typ reflect.Type) any {
	fnType := reflect.FuncOf([]reflect.Type{typ}, nil, false)

	return reflect.MakeFunc(fnType, func([]reflect.Value) []reflect.Value { return nil }).Interface()
}

// TestExecutorGraph validates the graph of each executor command without
// starting it and asserts that no control-plane type resolves in it: an
// executor reaches the server through ExecutorBackend alone. The probe is
// checked against the standalone server graph, where the types do resolve.
func TestExecutorGraph(t *testing.T) {
	unsetEnv(t, "KUBERNETES_SERVICE_HOST")

	t.Run("probe resolves in the standalone server graph", func(t *testing.T) {
		server := NewFxAppOptions(optionutil.ApplyOptions(serverDefaultOptions(), ServerCommandOptions(true)...))

		require.NoError(t, fx.ValidateApp(fx.Options(server, fx.Invoke(requestType(forbiddenExecutorTypes["*gorm.DB"])))))
	})

	for name, graph := range executorGraphs() {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, fx.ValidateApp(graph), "fx graph invalid")

			for typeName, typ := range forbiddenExecutorTypes {
				err := fx.ValidateApp(fx.Options(graph, fx.Invoke(requestType(typ))))
				assert.Error(t, err, "%s resolves in the %s graph", typeName, name)
			}
		})
	}
}
