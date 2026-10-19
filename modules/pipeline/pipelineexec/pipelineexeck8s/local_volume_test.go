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

func claimVolume(t *testing.T, podSpec corev1.PodSpec, name, claim string) {
	t.Helper()

	for _, v := range podSpec.Volumes {
		if v.Name == name {
			require.NotNil(t, v.PersistentVolumeClaim, "volume %q is not a PersistentVolumeClaim", name)
			assert.Equal(t, claim, v.PersistentVolumeClaim.ClaimName)
			assert.False(t, v.PersistentVolumeClaim.ReadOnly)

			return
		}
	}

	assert.Failf(t, "volume missing", "no %q volume in %+v", name, podSpec.Volumes)
}

func TestLaunchSync_MountsLocalVolumeClaimAtLocal(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)
	runner.config.LocalVolumeClaim = "synclet-local"

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-local-2",
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
	assert.Equal(t, sourceVolume, localMountName(t, podSpec, "destination"))
	claimVolume(t, podSpec, sourceVolume, "synclet-local")
}

func TestLaunchConnectorTask_MountsLocalVolumeClaimAtLocal(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)
	runner.config.LocalVolumeClaim = "synclet-local"

	jobName, err := runner.LaunchConnectorTask(context.Background(), ConnectorTaskOptions{
		TaskID:   "task-local-2",
		TaskType: "Check",
		Image:    "connector:latest",
		Config:   []byte(`{}`),
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec

	claimVolume(t, podSpec, localMountName(t, podSpec, "connector"), "synclet-local")
}
