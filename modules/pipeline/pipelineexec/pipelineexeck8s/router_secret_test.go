package pipelineexeck8s

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestK8sSyncWorker_SecretCarriesRouterCatalogAndSalt(t *testing.T) {
	jobID := uuid.New()
	catalog := []byte(`{"streams":[{"stream":{"name":"orders","namespace":"public","json_schema":{"type":"object","properties":{"email":{"type":"string"}}}},"sync_mode":"full_refresh","destination_sync_mode":"append","generation_id":0,"minimum_generation_id":0,"sync_id":0,"mappers":[{"hash":["email"]}]}]}`)
	salt := []byte("salt-1")

	backend := &mockBackend{claimResult: &pipelineexec.ClaimJobResult{
		Job:               &pipelineservice.Job{ID: jobID},
		ConnectionID:      uuid.New(),
		SourceImage:       "source:latest",
		DestImage:         "dest:latest",
		ConfiguredCatalog: catalog,
		MapperSalt:        salt,
	}}

	client := fake.NewSimpleClientset()
	logger := &logging.Logger{Delegate: noopDelegate{}}
	launcher := NewK8sSyncLauncherAdapter(newTestSyncRunner(client), pipelineservice.RuntimeDefaults{}, logger)

	worker := NewK8sSyncWorker(backend, launcher, "", "test-worker", logger)
	require.NoError(t, worker.Execute(context.Background()))
	require.Empty(t, backend.completions)

	secrets, err := client.CoreV1().Secrets("test-ns").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, secrets.Items, 1)

	secret := secrets.Items[0]

	assert.Equal(t, catalog, secret.Data["router-catalog"])
	assert.Equal(t, salt, secret.Data["mapper-salt"])
	assert.NotContains(t, string(secret.Data["source-catalog"]), "mappers")
}
