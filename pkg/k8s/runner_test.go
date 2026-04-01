package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newTestSyncRunner(client *fake.Clientset) *SyncRunner {
	return &SyncRunner{
		client:    client,
		namespace: "test-ns",
		config: Config{
			SyncletImage: "synclet:test",
			ServerAddr:   "http://synclet:8081",
		},
	}
}

func TestSyncOptions_PerContainerResources(t *testing.T) {
	opts := SyncOptions{
		SourceMemoryLimit:   1024 * 1024 * 1024, // 1Gi
		SourceCPULimit:      2.0,
		SourceMemoryRequest: 512 * 1024 * 1024, // 512Mi
		SourceCPURequest:    0.5,
		DestMemoryLimit:     2 * 1024 * 1024 * 1024, // 2Gi
		DestCPULimit:        4.0,
		DestMemoryRequest:   1024 * 1024 * 1024, // 1Gi
		DestCPURequest:      1.0,
	}

	assert.Equal(t, int64(1024*1024*1024), opts.SourceMemoryLimit)
	assert.InDelta(t, 2.0, opts.SourceCPULimit, 0.001)
	assert.Equal(t, int64(512*1024*1024), opts.SourceMemoryRequest)
	assert.InDelta(t, 0.5, opts.SourceCPURequest, 0.001)
	assert.Equal(t, int64(2*1024*1024*1024), opts.DestMemoryLimit)
	assert.InDelta(t, 4.0, opts.DestCPULimit, 0.001)
	assert.Equal(t, int64(1024*1024*1024), opts.DestMemoryRequest)
	assert.InDelta(t, 1.0, opts.DestCPURequest, 0.001)
}

func TestSyncOptions_SchedulingFields(t *testing.T) {
	opts := SyncOptions{
		Tolerations: []corev1.Toleration{
			{Key: "gpu", Operator: corev1.TolerationOpEqual, Value: "true", Effect: corev1.TaintEffectNoSchedule},
		},
		NodeSelector: map[string]string{"disktype": "ssd", "region": "us-east"},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{Key: "kubernetes.io/arch", Operator: corev1.NodeSelectorOpIn, Values: []string{"amd64"}},
							},
						},
					},
				},
			},
		},
	}

	require.Len(t, opts.Tolerations, 1)
	assert.Equal(t, "gpu", opts.Tolerations[0].Key)
	require.Len(t, opts.NodeSelector, 2)
	assert.Equal(t, "ssd", opts.NodeSelector["disktype"])
	require.NotNil(t, opts.Affinity)
	require.NotNil(t, opts.Affinity.NodeAffinity)
}

func TestBuildResourceRequirements(t *testing.T) {
	t.Run("limits only", func(t *testing.T) {
		reqs := buildResourceRequirements(1024*1024*1024, 2.0, 0, 0)
		require.NotNil(t, reqs.Limits)
		assert.Equal(t, int64(1024*1024*1024), reqs.Limits.Memory().Value())
		assert.Equal(t, int64(2000), reqs.Limits.Cpu().MilliValue())
		assert.Nil(t, reqs.Requests)
	})

	t.Run("requests only", func(t *testing.T) {
		reqs := buildResourceRequirements(0, 0, 512*1024*1024, 0.5)
		assert.Nil(t, reqs.Limits)
		require.NotNil(t, reqs.Requests)
		assert.Equal(t, int64(512*1024*1024), reqs.Requests.Memory().Value())
		assert.Equal(t, int64(500), reqs.Requests.Cpu().MilliValue())
	})

	t.Run("both limits and requests", func(t *testing.T) {
		reqs := buildResourceRequirements(2*1024*1024*1024, 4.0, 1024*1024*1024, 1.0)
		require.NotNil(t, reqs.Limits)
		require.NotNil(t, reqs.Requests)
		assert.Equal(t, int64(2*1024*1024*1024), reqs.Limits.Memory().Value())
		assert.Equal(t, int64(1024*1024*1024), reqs.Requests.Memory().Value())
	})

	t.Run("no resources", func(t *testing.T) {
		reqs := buildResourceRequirements(0, 0, 0, 0)
		assert.Nil(t, reqs.Limits)
		assert.Nil(t, reqs.Requests)
	})
}

func TestLaunchSync_SchedulingFields(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	tolerations := []corev1.Toleration{
		{Key: "workload", Operator: corev1.TolerationOpEqual, Value: "sync", Effect: corev1.TaintEffectNoSchedule},
	}
	nodeSelector := map[string]string{"pool": "connectors"}
	affinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
				{
					Weight: 100,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{Key: "instance-type", Operator: corev1.NodeSelectorOpIn, Values: []string{"m5.xlarge"}},
						},
					},
				},
			},
		},
	}

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "test-job-1",
		ConnectionID:  "test-conn-1",
		SourceID:      "src-1",
		DestinationID: "dest-1",
		SourceImage:   "source:latest",
		SourceConfig:  []byte(`{}`),
		DestImage:     "dest:latest",
		DestConfig:    []byte(`{}`),

		SourceMemoryLimit: 1024 * 1024 * 1024,
		SourceCPULimit:    1.0,
		DestMemoryLimit:   2 * 1024 * 1024 * 1024,
		DestCPULimit:      2.0,

		Tolerations:  tolerations,
		NodeSelector: nodeSelector,
		Affinity:     affinity,
	})
	require.NoError(t, err)
	require.NotEmpty(t, jobName)

	// Verify the created K8s Job.
	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec

	// Verify scheduling fields.
	require.Len(t, podSpec.Tolerations, 1)
	assert.Equal(t, "workload", podSpec.Tolerations[0].Key)
	assert.Equal(t, "sync", podSpec.Tolerations[0].Value)

	require.NotNil(t, podSpec.NodeSelector)
	assert.Equal(t, "connectors", podSpec.NodeSelector["pool"])

	require.NotNil(t, podSpec.Affinity)
	require.NotNil(t, podSpec.Affinity.NodeAffinity)

	// Verify per-container resources.
	// Containers: [orchestrator, source, destination]
	require.Len(t, podSpec.Containers, 3)

	sourceContainer := podSpec.Containers[1]
	assert.Equal(t, "source", sourceContainer.Name)
	require.NotNil(t, sourceContainer.Resources.Limits)
	assert.Equal(t, int64(1024*1024*1024), sourceContainer.Resources.Limits.Memory().Value())
	assert.Equal(t, int64(1000), sourceContainer.Resources.Limits.Cpu().MilliValue())

	destContainer := podSpec.Containers[2]
	assert.Equal(t, "destination", destContainer.Name)
	require.NotNil(t, destContainer.Resources.Limits)
	assert.Equal(t, int64(2*1024*1024*1024), destContainer.Resources.Limits.Memory().Value())
	assert.Equal(t, int64(2000), destContainer.Resources.Limits.Cpu().MilliValue())
}

func TestLaunchSync_NoSchedulingFields(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "test-job-2",
		ConnectionID:  "test-conn-2",
		SourceID:      "src-2",
		DestinationID: "dest-2",
		SourceImage:   "source:latest",
		SourceConfig:  []byte(`{}`),
		DestImage:     "dest:latest",
		DestConfig:    []byte(`{}`),
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec

	// No scheduling fields when not set.
	assert.Nil(t, podSpec.Tolerations)
	assert.Nil(t, podSpec.NodeSelector)
	assert.Nil(t, podSpec.Affinity)

	// No resource limits when not set.
	sourceContainer := podSpec.Containers[1]
	assert.Nil(t, sourceContainer.Resources.Limits)
	assert.Nil(t, sourceContainer.Resources.Requests)
}

func TestK8sSecurityContext(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "test-job-sec",
		ConnectionID:  "test-conn-sec",
		SourceID:      "src-sec",
		DestinationID: "dest-sec",
		SourceImage:   "source:latest",
		SourceConfig:  []byte(`{}`),
		DestImage:     "dest:latest",
		DestConfig:    []byte(`{}`),
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec
	require.Len(t, podSpec.Containers, 3)

	// Orchestrator container: fully locked down.
	orchContainer := podSpec.Containers[0]
	assert.Equal(t, "orchestrator", orchContainer.Name)
	assert.Equal(t, corev1.PullIfNotPresent, orchContainer.ImagePullPolicy, "orchestrator uses PullIfNotPresent")
	require.NotNil(t, orchContainer.SecurityContext, "orchestrator must have SecurityContext")
	assert.True(t, *orchContainer.SecurityContext.RunAsNonRoot, "orchestrator must run as non-root")
	assert.Equal(t, int64(1000), *orchContainer.SecurityContext.RunAsUser, "orchestrator must run as user 1000")
	assert.False(t, *orchContainer.SecurityContext.AllowPrivilegeEscalation, "orchestrator must not allow privilege escalation")
	assert.True(t, *orchContainer.SecurityContext.ReadOnlyRootFilesystem, "orchestrator must have read-only root fs")
	require.NotNil(t, orchContainer.SecurityContext.Capabilities)
	assert.Contains(t, orchContainer.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"))

	// Source container: drop capabilities, no privilege escalation, but no RunAsNonRoot.
	sourceContainer := podSpec.Containers[1]
	assert.Equal(t, "source", sourceContainer.Name)
	assert.Equal(t, corev1.PullAlways, sourceContainer.ImagePullPolicy, "source must use PullAlways for digest security")
	require.NotNil(t, sourceContainer.SecurityContext, "source must have SecurityContext")
	assert.False(t, *sourceContainer.SecurityContext.AllowPrivilegeEscalation, "source must not allow privilege escalation")
	require.NotNil(t, sourceContainer.SecurityContext.Capabilities)
	assert.Contains(t, sourceContainer.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"))
	assert.Nil(t, sourceContainer.SecurityContext.RunAsNonRoot, "source must not enforce runAsNonRoot")
	assert.Nil(t, sourceContainer.SecurityContext.ReadOnlyRootFilesystem, "source must not enforce readOnlyRootFilesystem")

	// Destination container: same as source.
	destContainer := podSpec.Containers[2]
	assert.Equal(t, "destination", destContainer.Name)
	assert.Equal(t, corev1.PullAlways, destContainer.ImagePullPolicy, "destination must use PullAlways for digest security")
	require.NotNil(t, destContainer.SecurityContext, "destination must have SecurityContext")
	assert.False(t, *destContainer.SecurityContext.AllowPrivilegeEscalation, "destination must not allow privilege escalation")
	require.NotNil(t, destContainer.SecurityContext.Capabilities)
	assert.Contains(t, destContainer.SecurityContext.Capabilities.Drop, corev1.Capability("ALL"))
	assert.Nil(t, destContainer.SecurityContext.RunAsNonRoot, "destination must not enforce runAsNonRoot")
	assert.Nil(t, destContainer.SecurityContext.ReadOnlyRootFilesystem, "destination must not enforce readOnlyRootFilesystem")
}

func TestLaunchSync_PerContainerResourceRequests(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "test-job-3",
		ConnectionID:  "test-conn-3",
		SourceID:      "src-3",
		DestinationID: "dest-3",
		SourceImage:   "source:latest",
		SourceConfig:  []byte(`{}`),
		DestImage:     "dest:latest",
		DestConfig:    []byte(`{}`),

		SourceMemoryRequest: 256 * 1024 * 1024,
		SourceCPURequest:    0.25,
		DestMemoryRequest:   512 * 1024 * 1024,
		DestCPURequest:      0.5,
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	podSpec := job.Spec.Template.Spec

	sourceContainer := podSpec.Containers[1]
	require.NotNil(t, sourceContainer.Resources.Requests)
	assert.Equal(t, int64(256*1024*1024), sourceContainer.Resources.Requests.Memory().Value())
	assert.True(t, sourceContainer.Resources.Requests.Cpu().Equal(resource.MustParse("250m")))

	destContainer := podSpec.Containers[2]
	require.NotNil(t, destContainer.Resources.Requests)
	assert.Equal(t, int64(512*1024*1024), destContainer.Resources.Requests.Memory().Value())
	assert.True(t, destContainer.Resources.Requests.Cpu().Equal(resource.MustParse("500m")))
}

func TestLaunchSync_CreatesSecret(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	sourceConfig := []byte(`{"host":"db.example.com","password":"s3cret"}`)
	destConfig := []byte(`{"token":"abc123"}`)
	catalog := []byte(`{"streams":[]}`)
	state := []byte(`{"cursor":"2024-01-01"}`)

	_, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-secret-1",
		ConnectionID:  "conn-1",
		SourceID:      "src-1",
		DestinationID: "dest-1",
		SourceImage:   "source:latest",
		SourceConfig:  sourceConfig,
		DestImage:     "dest:latest",
		DestConfig:    destConfig,
		SourceCatalog: catalog,
		DestCatalog:   catalog,
		State:         state,
	})
	require.NoError(t, err)

	// Verify Secret was created with expected data keys.
	secretName := sanitizeK8sName("synclet-sync-" + "job-secret-1")
	secret, err := client.CoreV1().Secrets("test-ns").Get(context.Background(), secretName, metav1.GetOptions{})
	require.NoError(t, err)

	assert.Equal(t, sourceConfig, secret.Data["source-config"])
	assert.Equal(t, destConfig, secret.Data["dest-config"])
	assert.Equal(t, catalog, secret.Data["source-catalog"])
	assert.Equal(t, catalog, secret.Data["dest-catalog"])
	assert.Equal(t, state, secret.Data["source-state"])

	// Verify labels.
	assert.Equal(t, "synclet", secret.Labels["app"])
	assert.Equal(t, "job-secret-1", secret.Labels["synclet.io/sync-job"])
}

func TestLaunchSync_SecretVolume(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-vol-1",
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

	// Verify sync-configs volume exists with SecretVolumeSource.
	var secretVol *corev1.Volume

	for i := range podSpec.Volumes {
		if podSpec.Volumes[i].Name == "sync-configs" {
			secretVol = &podSpec.Volumes[i]

			break
		}
	}

	require.NotNil(t, secretVol, "sync-configs volume must exist")
	require.NotNil(t, secretVol.Secret, "sync-configs must be a Secret volume")

	secretName := sanitizeK8sName("synclet-sync-" + "job-vol-1")
	assert.Equal(t, secretName, secretVol.Secret.SecretName)

	// Verify orchestrator container has the volume mount.
	orchContainer := podSpec.Containers[0]
	assert.Equal(t, "orchestrator", orchContainer.Name)
	var secretMount *corev1.VolumeMount

	for i := range orchContainer.VolumeMounts {
		if orchContainer.VolumeMounts[i].Name == "sync-configs" {
			secretMount = &orchContainer.VolumeMounts[i]

			break
		}
	}

	require.NotNil(t, secretMount, "orchestrator must have sync-configs mount")
	assert.Equal(t, "/secrets", secretMount.MountPath)
	assert.True(t, secretMount.ReadOnly)
}

func TestLaunchSync_DualPathArgs(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	sourceConfig := []byte(`{"host":"db"}`)
	destConfig := []byte(`{"token":"x"}`)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-dual-1",
		ConnectionID:  "conn-1",
		SourceID:      "src-1",
		DestinationID: "dest-1",
		SourceImage:   "source:latest",
		SourceConfig:  sourceConfig,
		DestImage:     "dest:latest",
		DestConfig:    destConfig,
	})
	require.NoError(t, err)

	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	orchArgs := job.Spec.Template.Spec.Containers[0].Command

	// Must NOT contain base64-encoded config args — configs are in K8s Secret.
	assert.NotContains(t, orchArgs, "--source-config")
	assert.NotContains(t, orchArgs, "--dest-config")

	// Must contain --secrets-dir /secrets.
	assert.Contains(t, orchArgs, "--secrets-dir")
	assert.Contains(t, orchArgs, "/secrets")
}

func TestLaunchSync_SecretsDir(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-sdir-1",
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

	orchArgs := job.Spec.Template.Spec.Containers[0].Command

	// Find --secrets-dir and verify the next arg is /secrets.
	found := false

	for i, arg := range orchArgs {
		if arg == "--secrets-dir" && i+1 < len(orchArgs) {
			assert.Equal(t, "/secrets", orchArgs[i+1])

			found = true

			break
		}
	}

	assert.True(t, found, "orchestrator args must contain --secrets-dir /secrets")
}

func TestLaunchSync_CleansSecretOnFailure(t *testing.T) {
	client := fake.NewSimpleClientset()

	// Inject job creation failure.
	client.PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("simulated job creation failure")
	})

	runner := newTestSyncRunner(client)

	_, err := runner.LaunchSync(context.Background(), SyncOptions{
		JobID:         "job-fail-1",
		ConnectionID:  "conn-1",
		SourceID:      "src-1",
		DestinationID: "dest-1",
		SourceImage:   "source:latest",
		SourceConfig:  []byte(`{}`),
		DestImage:     "dest:latest",
		DestConfig:    []byte(`{}`),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated job creation failure")

	// Verify the Secret was cleaned up (should not exist).
	secretName := sanitizeK8sName("synclet-sync-" + "job-fail-1")
	_, err = client.CoreV1().Secrets("test-ns").Get(context.Background(), secretName, metav1.GetOptions{})
	assert.Error(t, err, "Secret should have been deleted after job creation failure")
}

func TestLaunchConnectorTask_CreatesSecret(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	taskConfig := []byte(`{"api_key":"secret123"}`)

	jobName, err := runner.LaunchConnectorTask(context.Background(), ConnectorTaskOptions{
		TaskID:         "task-1",
		TaskType:       "Check",
		Image:          "connector:latest",
		Config:         taskConfig,
		InternalAPIURL: "http://synclet:8081",
	})
	require.NoError(t, err)
	require.NotEmpty(t, jobName)

	// Find the secret -- it should start with "synclet-task-" and NOT equal the jobName.
	secrets, err := client.CoreV1().Secrets("test-ns").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, secrets.Items, 1)

	secret := secrets.Items[0]
	assert.True(t, strings.HasPrefix(secret.Name, "synclet-task-"), "secret name must start with synclet-task-")
	assert.Contains(t, secret.Name, "check", "secret name must contain task type")
	assert.Equal(t, taskConfig, secret.Data["task-config"])
	assert.Equal(t, "task-1", secret.Labels["synclet.io/task"])

	// Verify coordinator args contain --secrets-dir but NOT --task-config (config is in Secret).
	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	coordArgs := job.Spec.Template.Spec.Containers[0].Command
	assert.NotContains(t, coordArgs, "--task-config")
	assert.Contains(t, coordArgs, "--secrets-dir")
	assert.Contains(t, coordArgs, "/secrets")

	// Verify volume mount exists.
	var secretMount *corev1.VolumeMount

	for i := range job.Spec.Template.Spec.Containers[0].VolumeMounts {
		if job.Spec.Template.Spec.Containers[0].VolumeMounts[i].Name == "task-configs" {
			secretMount = &job.Spec.Template.Spec.Containers[0].VolumeMounts[i]

			break
		}
	}

	require.NotNil(t, secretMount, "coordinator must have task-configs mount")
	assert.Equal(t, "/secrets", secretMount.MountPath)
	assert.True(t, secretMount.ReadOnly)
}

func TestLaunchConnectorTask_SpecNoSecret(t *testing.T) {
	client := fake.NewSimpleClientset()
	runner := newTestSyncRunner(client)

	jobName, err := runner.LaunchConnectorTask(context.Background(), ConnectorTaskOptions{
		TaskID:         "task-spec-1",
		TaskType:       "Spec",
		Image:          "connector:latest",
		Config:         nil, // No config for spec tasks.
		InternalAPIURL: "http://synclet:8081",
	})
	require.NoError(t, err)
	require.NotEmpty(t, jobName)

	// No secret should be created.
	secrets, err := client.CoreV1().Secrets("test-ns").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, secrets.Items, "no Secret should be created for spec tasks with nil config")

	// No --secrets-dir arg.
	job, err := client.BatchV1().Jobs("test-ns").Get(context.Background(), jobName, metav1.GetOptions{})
	require.NoError(t, err)

	coordArgs := job.Spec.Template.Spec.Containers[0].Command
	assert.NotContains(t, coordArgs, "--secrets-dir")
}
