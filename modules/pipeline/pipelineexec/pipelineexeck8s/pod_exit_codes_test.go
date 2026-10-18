package pipelineexeck8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testConnectionID = "44444444-4444-4444-4444-444444444444"

func createSyncPod(t *testing.T, client *fake.Clientset, jobID string, status corev1.PodStatus) {
	t.Helper()

	ctx := context.Background()

	_, err := client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "k8s-" + jobID,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-30 * time.Minute)),
			Labels: map[string]string{
				pipelineexec.OwnerLabel: testOwner,
				"synclet.io/sync-job":   jobID,
				"synclet.io/connection": testConnectionID,
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-" + jobID + "-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "k8s-" + jobID},
		},
		Status: status,
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

func terminatedContainer(name, reason string, exitCode int32) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name:  name,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: reason, ExitCode: exitCode}},
	}
}

func runningContainer(name string) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name:  name,
		State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
	}
}

func TestReconcileSyncJob_FailedPodReportsConnectorExitCodes(t *testing.T) {
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{activeJobs: map[string]bool{jobID1: true}}

	createSyncPod(t, client, jobID1, corev1.PodStatus{
		Phase: corev1.PodFailed,
		ContainerStatuses: []corev1.ContainerStatus{
			terminatedContainer("orchestrator", "Error", 2),
			terminatedContainer("source", "OOMKilled", 137),
			terminatedContainer("destination", "Error", 1),
		},
	})

	newTestReconciler(client, checker).Reconcile(context.Background(), true)

	require.Len(t, checker.completions, 1)
	completion := checker.completions[0]
	assert.Equal(t, jobID1, completion.JobID.String())
	assert.Equal(t, testConnectionID, completion.ConnectionID.String())
	assert.False(t, completion.Success)
	assert.Equal(t, 137, completion.SourceExitCode)
	assert.Equal(t, 1, completion.DestExitCode)
}

func TestReconcileSyncJob_OOMKilledSourceInRunningPodFailsJob(t *testing.T) {
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{activeJobs: map[string]bool{jobID2: true}}

	createSyncPod(t, client, jobID2, corev1.PodStatus{
		Phase: corev1.PodRunning,
		ContainerStatuses: []corev1.ContainerStatus{
			runningContainer("orchestrator"),
			terminatedContainer("source", "OOMKilled", 0),
			runningContainer("destination"),
		},
	})

	newTestReconciler(client, checker).Reconcile(context.Background(), true)

	require.Len(t, checker.completions, 1)
	assert.Contains(t, checker.completions[0].ErrorMessage, "OOMKilled")
	assert.Equal(t, 137, checker.completions[0].SourceExitCode)
	assert.Equal(t, 0, checker.completions[0].DestExitCode)
}
