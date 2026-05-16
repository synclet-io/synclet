package pipelineexeck8s

import (
	"context"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// mockResourceChecker records calls for verification.
type mockResourceChecker struct {
	activeJobs   map[string]bool
	activeTasks  map[string]bool
	failJobCalls []failJobCall
}

type failJobCall struct {
	JobID  string
	Reason string
}

func (m *mockResourceChecker) IsJobActive(_ context.Context, jobID string) (bool, error) {
	if m.activeJobs != nil {
		return m.activeJobs[jobID], nil
	}

	return false, nil
}

func (m *mockResourceChecker) IsTaskActive(_ context.Context, taskID string) (bool, error) {
	if m.activeTasks != nil {
		return m.activeTasks[taskID], nil
	}

	return false, nil
}

func (m *mockResourceChecker) FailJob(_ context.Context, jobID, reason string) error {
	m.failJobCalls = append(m.failJobCalls, failJobCall{JobID: jobID, Reason: reason})

	return nil
}

// noopDelegate is a no-op logging delegate for tests.
type noopDelegate struct{}

func (noopDelegate) Info(context.Context, string, ...interface{})         {}
func (noopDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopDelegate) Named(string) logging.Delegate                      { return d }
func (d noopDelegate) SkipCallers(int) logging.Delegate                   { return d }

func newTestReconciler(client *fake.Clientset, checker *mockResourceChecker) *ResourceCleaner {
	return NewReconciler(client, "default", checker, &logging.Logger{Delegate: noopDelegate{}})
}

func TestReconcileSyncJob_ImagePullBackOff(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{"job-1": true},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	// Create K8s Job with sync-job label.
	_, err := client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "k8s-job-1",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "job-1"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create pod with ImagePullBackOff status.
	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-job-1-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "k8s-job-1"},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "source",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	// Verify FailJob was called with ImagePullBackOff reason.
	require.Len(t, checker.failJobCalls, 1)
	assert.Equal(t, "job-1", checker.failJobCalls[0].JobID)
	assert.Contains(t, checker.failJobCalls[0].Reason, "ImagePullBackOff")
	assert.Contains(t, checker.failJobCalls[0].Reason, "source")

	// Verify K8s job was deleted.
	_, err = client.BatchV1().Jobs("default").Get(ctx, "k8s-job-1", metav1.GetOptions{})
	assert.Error(t, err, "K8s job should have been deleted")
}

func TestReconcileSyncJob_ErrImagePull(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{"job-2": true},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	_, err := client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "k8s-job-2",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "job-2"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-job-2-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "k8s-job-2"},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "destination",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ErrImagePull",
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	require.Len(t, checker.failJobCalls, 1)
	assert.Equal(t, "job-2", checker.failJobCalls[0].JobID)
	assert.Contains(t, checker.failJobCalls[0].Reason, "ErrImagePull")
	assert.Contains(t, checker.failJobCalls[0].Reason, "destination")

	_, err = client.BatchV1().Jobs("default").Get(ctx, "k8s-job-2", metav1.GetOptions{})
	assert.Error(t, err, "K8s job should have been deleted")
}

func TestReconcileSyncJob_InitContainerImagePullBackOff(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{"job-3": true},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	_, err := client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "k8s-job-3",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "job-3"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-job-3-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "k8s-job-3"},
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init-sidecar",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	require.Len(t, checker.failJobCalls, 1)
	assert.Contains(t, checker.failJobCalls[0].Reason, "ImagePullBackOff")
	assert.Contains(t, checker.failJobCalls[0].Reason, "init-sidecar")
}

func TestReconciler_OrphanSyncJobCleanup(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{"active-job": true},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	// Create an orphan K8s job (DB job not active, old enough).
	_, err := client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "orphan-k8s-job",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "orphan-job-id"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a running pod for the orphan job.
	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-k8s-job-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "orphan-k8s-job"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create an active K8s job (DB job active, old enough).
	_, err = client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "active-k8s-job",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "active-job"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = client.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "active-k8s-job-pod",
			Namespace: "default",
			Labels:    map[string]string{"synclet.io/job": "active-k8s-job"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create a recent K8s job (within grace period — should NOT be cleaned up).
	_, err = client.BatchV1().Jobs("default").Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "recent-k8s-job",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels:            map[string]string{"synclet.io/sync-job": "recent-job-id"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	// Orphan job should be deleted.
	_, err = client.BatchV1().Jobs("default").Get(ctx, "orphan-k8s-job", metav1.GetOptions{})
	require.Error(t, err, "orphan K8s job should have been deleted")

	// Active job should still exist.
	_, err = client.BatchV1().Jobs("default").Get(ctx, "active-k8s-job", metav1.GetOptions{})
	require.NoError(t, err, "active K8s job should NOT have been deleted")

	// Recent job should still exist (grace period).
	_, err = client.BatchV1().Jobs("default").Get(ctx, "recent-k8s-job", metav1.GetOptions{})
	assert.NoError(t, err, "recent K8s job should NOT have been deleted")
}

func TestReconciler_CleansOrphanedSyncSecrets(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	_, err := client.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "synclet-sync-job-1",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "job-1"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	_, err = client.CoreV1().Secrets("default").Get(ctx, "synclet-sync-job-1", metav1.GetOptions{})
	assert.Error(t, err, "orphan sync secret should have been deleted")
}

func TestReconciler_KeepsSyncSecretForActiveJob(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{"job-2": true},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	_, err := client.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "synclet-sync-job-2",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/sync-job": "job-2"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	_, err = client.CoreV1().Secrets("default").Get(ctx, "synclet-sync-job-2", metav1.GetOptions{})
	assert.NoError(t, err, "sync secret for active job should NOT have been deleted")
}

func TestReconciler_CleansOrphanedTaskSecrets(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeTasks: map[string]bool{},
	}

	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))

	_, err := client.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "synclet-task-check-123",
			Namespace:         "default",
			CreationTimestamp: oldTime,
			Labels:            map[string]string{"synclet.io/task": "task-1"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	r := newTestReconciler(client, checker)
	r.Reconcile(ctx, true)

	_, err = client.CoreV1().Secrets("default").Get(ctx, "synclet-task-check-123", metav1.GetOptions{})
	assert.Error(t, err, "orphan task secret should have been deleted")
}

func TestReconciler_GracePeriodApplies(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	checker := &mockResourceChecker{
		activeJobs: map[string]bool{},
	}

	// Create a recent sync Secret (within grace period).
	_, err := client.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "synclet-sync-recent",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels:            map[string]string{"synclet.io/sync-job": "recent-job"},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	rec := newTestReconciler(client, checker)

	// Reconcile with grace should NOT delete the recent secret.
	rec.Reconcile(ctx, true)
	_, err = client.CoreV1().Secrets("default").Get(ctx, "synclet-sync-recent", metav1.GetOptions{})
	require.NoError(t, err, "recent sync secret should NOT have been deleted (grace period)")

	// Reconcile without grace (startup) SHOULD delete it.
	rec.Reconcile(ctx, false)
	_, err = client.CoreV1().Secrets("default").Get(ctx, "synclet-sync-recent", metav1.GetOptions{})
	assert.Error(t, err, "recent sync secret should have been deleted by startup reconcile")
}
