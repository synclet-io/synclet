package app

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/go-pnp/go-pnp/pkg/ordering"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
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
	"*pipelinejobs.RegisterExecutor":   reflect.TypeOf((*pipelinejobs.RegisterExecutor)(nil)),
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

// unreachableKubeconfig is a kubeconfig whose API server nothing listens on:
// building a Kubernetes client from it never dials.
const unreachableKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: none
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: none
  context:
    cluster: none
    user: none
current-context: none
users:
- name: none
  user:
    token: none
`

// TestExecutorRegistrationJob asserts that every graph running an executor,
// the standalone server included, wires up the executor_registration job.
// The graphs are built, not started: a job decorator records the name of each
// job pnpjobber wires up. The database connection is replaced by one that
// never dials and the Kubernetes client points at an unreachable server.
func TestExecutorRegistrationJob(t *testing.T) {
	unsetEnv(t, "KUBERNETES_SERVICE_HOST")
	t.Setenv("AUTH_JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("SECRET_ENCRYPTION_KEY", validEncryptionKey)

	for key, value := range envDefaults {
		if os.Getenv(key) == "" {
			t.Setenv(key, value)
		}
	}

	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(unreachableKubeconfig), 0o600))
	t.Setenv("K8S_EXECUTOR_KUBECONFIG", kubeconfig)

	db, err := gorm.Open(postgres.Open("postgres://synclet@127.0.0.1:1/synclet"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)

	graphs := executorGraphs()
	graphs["standalone server"] = fx.Options(
		NewFxAppOptions(optionutil.ApplyOptions(serverDefaultOptions(), ServerCommandOptions(true)...)),
		fx.Replace(db),
	)

	for name, graph := range graphs {
		t.Run(name, func(t *testing.T) {
			var jobs []string

			recordJobs := func() ordering.OrderedItem[pnpjobber.JobDecorator] {
				return ordering.Ordered[pnpjobber.JobDecorator](0, func(job jobber.Job) jobber.Job {
					jobs = append(jobs, job.Name())

					return job
				})
			}

			app := fx.New(fx.NopLogger, graph, fx.Provide(pnpjobber.JobDecoratorProvider(recordJobs)))
			require.NoError(t, app.Err())

			assert.Contains(t, jobs, "executor_registration")
		})
	}
}

// TestExecutorInfoResolves asserts that every graph running an executor,
// the standalone server included, describes it to the executor registry.
func TestExecutorInfoResolves(t *testing.T) {
	unsetEnv(t, "KUBERNETES_SERVICE_HOST")

	executorInfo := reflect.TypeOf(pipelineexec.ExecutorInfo{})
	graphs := executorGraphs()
	graphs["standalone server"] = NewFxAppOptions(optionutil.ApplyOptions(serverDefaultOptions(), ServerCommandOptions(true)...))

	for name, graph := range graphs {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, fx.ValidateApp(fx.Options(graph, fx.Invoke(requestType(executorInfo)))))
		})
	}
}
