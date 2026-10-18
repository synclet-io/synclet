package pipelineadapt

import (
	"context"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexeck8s"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

// activeJobChecker reports every job and task as active and completes jobs
// through the use case backend.
type activeJobChecker struct {
	*UseCaseExecutorBackend
}

func (activeJobChecker) IsJobActive(context.Context, string) (bool, error)  { return true, nil }
func (activeJobChecker) IsTaskActive(context.Context, string) (bool, error) { return true, nil }

func TestK8sReconciler_OOMKilledSourcePodFailsJobWithoutRetry(t *testing.T) {
	ctx := t.Context()
	logger := &logging.Logger{Delegate: noopLogDelegate{}}
	storage := &completionStorage{job: &pipelineservice.Job{
		ID:          uuid.New(),
		Status:      pipelineservice.JobStatusRunning,
		Attempt:     1,
		MaxAttempts: 3,
	}}
	checker := activeJobChecker{&UseCaseExecutorBackend{
		reportCompletion: pipelinejobs.NewReportCompletion(pipelinejobs.NewUpdateJobStatus(storage), storage, nil, logger, pipelineservice.Config{}),
	}}

	client := fake.NewSimpleClientset()
	_, err := client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "synclet-sync-1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
			Labels: map[string]string{
				pipelineexec.OwnerLabel: "executor-0",
				"synclet.io/sync-job":   storage.job.ID.String(),
				"synclet.io/connection": uuid.NewString(),
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "synclet-sync-1-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "synclet-sync-1"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "source", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}}},
				{Name: "destination", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"}}},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	pipelineexeck8s.NewReconciler(client, "default", "executor-0", checker, logger).Reconcile(ctx, true)

	require.NotNil(t, storage.written)
	assert.Equal(t, pipelineservice.JobStatusFailed, storage.written.Status)
	assert.Equal(t, 1, storage.written.Attempt)
	require.NotNil(t, storage.written.FailureReason)
	assert.Equal(t, "OOMKilled (exit 137)", *storage.written.FailureReason)
}
