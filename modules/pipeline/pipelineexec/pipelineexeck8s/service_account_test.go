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

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestLaunchSyncJob_ServiceAccountName(t *testing.T) {
	tests := []struct {
		name           string
		defaultAccount string
		srcRuntime     string
		destRuntime    string
		want           string
	}{
		{name: "source override wins", defaultAccount: "global", srcRuntime: `{"service_account_name":"src"}`, destRuntime: `{"service_account_name":"dest"}`, want: "src"},
		{name: "destination override when source has none", defaultAccount: "global", srcRuntime: `{"cpu_limit":"1"}`, destRuntime: `{"service_account_name":"dest"}`, want: "dest"},
		{name: "destination override without source runtime config", defaultAccount: "global", destRuntime: `{"service_account_name":"dest"}`, want: "dest"},
		{name: "global default", defaultAccount: "global", want: "global"},
		{name: "none when all are empty", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			adapter := NewK8sSyncLauncherAdapter(newTestSyncRunner(client), pipelineservice.RuntimeDefaults{ServiceAccountName: tt.defaultAccount}, &logging.Logger{Delegate: noopDelegate{}})

			jobName, err := adapter.LaunchSyncJob(context.Background(), K8sSyncJobOptions{
				JobID:             uuid.New(),
				ConnectionID:      uuid.New(),
				SourceID:          uuid.New(),
				DestinationID:     uuid.New(),
				SourceImage:       "source:latest",
				SourceConfig:      []byte(`{}`),
				DestImage:         "dest:latest",
				DestConfig:        []byte(`{}`),
				RuntimeConfig:     tt.srcRuntime,
				DestRuntimeConfig: tt.destRuntime,
			})
			require.NoError(t, err)

			job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, job.Spec.Template.Spec.ServiceAccountName)
		})
	}
}

func TestLaunchTask_ServiceAccountName(t *testing.T) {
	for _, taskType := range []pipelineservice.ConnectorTaskType{
		pipelineservice.ConnectorTaskTypeCheck,
		pipelineservice.ConnectorTaskTypeSpec,
		pipelineservice.ConnectorTaskTypeDiscover,
	} {
		for _, defaultAccount := range []string{"global", ""} {
			t.Run(taskType.String()+"/"+defaultAccount, func(t *testing.T) {
				client := fake.NewSimpleClientset()
				adapter := NewK8sTaskLauncherAdapter(newTestSyncRunner(client), pipelineservice.RuntimeDefaults{ServiceAccountName: defaultAccount})

				jobName, err := adapter.LaunchTask(context.Background(), TaskOptions{
					TaskID:   uuid.NewString(),
					TaskType: taskType,
					Image:    "connector:latest",
					Config:   []byte(`{}`),
				})
				require.NoError(t, err)

				job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, defaultAccount, job.Spec.Template.Spec.ServiceAccountName)
			})
		}
	}
}
