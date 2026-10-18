package pipelineexeck8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func emptyDirVolume(t *testing.T, podSpec corev1.PodSpec, name string) {
	t.Helper()

	for _, v := range podSpec.Volumes {
		if v.Name == name {
			assert.NotNil(t, v.EmptyDir, "volume %q is not an emptyDir", name)

			return
		}
	}

	assert.Failf(t, "volume missing", "no %q volume in %+v", name, podSpec.Volumes)
}

func localMountName(t *testing.T, podSpec corev1.PodSpec, containerName string) string {
	t.Helper()

	for _, c := range podSpec.Containers {
		if c.Name != containerName {
			continue
		}

		for _, m := range c.VolumeMounts {
			if m.MountPath == "/local" {
				assert.False(t, m.ReadOnly, "/local of %s is read-only", containerName)

				return m.Name
			}
		}

		assert.Failf(t, "no /local mount", "container %s mounts %+v", containerName, c.VolumeMounts)

		return ""
	}

	require.Failf(t, "container missing", "no container %q", containerName)

	return ""
}

func TestLaunchSync_MountsEmptyDirAtLocal(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-local-1",
		ConnectionID:  "conn-1",
		SourceID:      "src-1",
		DestinationID: "dest-1",
		SourceImage:   "source:latest",
		SourceConfig:  []byte(`{}`),
		DestImage:     "dest:latest",
		DestConfig:    []byte(`{}`),
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec

	sourceVolume := localMountName(t, podSpec, "source")
	destVolume := localMountName(t, podSpec, "destination")
	assert.Equal(t, sourceVolume, destVolume, "source and destination share the job's /local")
	emptyDirVolume(t, podSpec, sourceVolume)
}

func TestLaunchConnectorTask_MountsEmptyDirAtLocal(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchConnectorTask(context.Background(), ConnectorTaskOptions{
		TaskID:   "task-local-1",
		TaskType: "Check",
		Image:    "connector:latest",
		Config:   []byte(`{}`),
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec

	emptyDirVolume(t, podSpec, localMountName(t, podSpec, "connector"))
}
