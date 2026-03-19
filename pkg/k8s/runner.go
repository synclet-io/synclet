package k8s

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds Kubernetes runner configuration.
type Config struct {
	Namespace       string
	Kubeconfig      string
	ImagePullSecret string
	DefaultMemory   string
	DefaultCPU      string
	SyncletImage    string // Docker image containing the synclet binary (for orchestrator)
	ServerAddr      string // Synclet server address reachable from pods
}

// SyncOptions contains all parameters needed to launch a sync pod.
type SyncOptions struct {
	JobID        string
	ConnectionID string
	JobName      string

	SourceID      string // UUID of source entity
	DestinationID string // UUID of destination entity

	SourceImage   string
	SourceConfig  []byte
	DestImage     string
	DestConfig    []byte
	SourceCatalog []byte // JSON configured catalog for source (original namespaces)
	DestCatalog   []byte // JSON configured catalog for destination (namespace-rewritten)
	State         []byte

	// Namespace/prefix rewriting settings for the coordinator's message router.
	NamespaceDefinition   string // "source", "destination", "custom"
	CustomNamespaceFormat string // Template with ${SOURCE_NAMESPACE} placeholder
	StreamPrefix          string

	// Per-container resource limits (source container).
	SourceMemoryLimit   int64
	SourceCPULimit      float64
	SourceMemoryRequest int64
	SourceCPURequest    float64

	// Per-container resource limits (destination container).
	DestMemoryLimit   int64
	DestCPULimit      float64
	DestMemoryRequest int64
	DestCPURequest    float64

	// K8s scheduling (pod-level, applies to entire pod).
	Tolerations        []corev1.Toleration
	NodeSelector       map[string]string
	Affinity           *corev1.Affinity
	ServiceAccountName string
}

// SyncRunner launches and manages K8s sync pods with the 3-container architecture.
type SyncRunner struct {
	client    kubernetes.Interface
	namespace string
	config    Config
}

// NewSyncRunner creates a new K8s sync runner.
func NewSyncRunner(cfg Config) (*SyncRunner, error) {
	var restConfig *rest.Config
	var err error

	if cfg.Kubeconfig != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	} else {
		restConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("creating k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating k8s client: %w", err)
	}

	ns := cfg.Namespace
	if ns == "" {
		// When running in-cluster, use the pod's own namespace.
		if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			ns = strings.TrimSpace(string(data))
		}
	}
	if ns == "" {
		ns = "default"
	}

	return &SyncRunner{
		client:    clientset,
		namespace: ns,
		config:    cfg,
	}, nil
}

// Client returns the underlying Kubernetes client for use by the reconciler.
func (r *SyncRunner) Client() kubernetes.Interface {
	return r.client
}

// Namespace returns the configured namespace.
func (r *SyncRunner) Namespace() string {
	return r.namespace
}

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }

// LaunchSync creates a K8s Job with a 3-container pod (orchestrator + source + dest).
// It returns immediately; the orchestrator inside the pod handles the sync lifecycle.
func (r *SyncRunner) LaunchSync(ctx context.Context, opts SyncOptions) (string, error) {
	jobName := opts.JobName
	if jobName == "" {
		jobName = fmt.Sprintf("synclet-sync-%d", time.Now().UnixNano())
	}
	jobName = sanitizeK8sName(jobName)

	dataDir := "/shared"

	// Create a K8s Secret with connector config data.
	// Secrets keep sensitive credentials (passwords, API keys) out of CLI args
	// which are visible in `kubectl describe pod` and K8s audit logs.
	secretName := sanitizeK8sName(fmt.Sprintf("synclet-sync-%s", opts.JobID))
	secretData := map[string][]byte{
		"source-config": opts.SourceConfig,
		"dest-config":   opts.DestConfig,
	}
	if opts.SourceCatalog != nil {
		secretData["source-catalog"] = opts.SourceCatalog
	}
	if opts.DestCatalog != nil {
		secretData["dest-catalog"] = opts.DestCatalog
	}
	if opts.State != nil {
		secretData["source-state"] = opts.State
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"app":                 "synclet",
				"synclet.io/sync-job": opts.JobID,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
	if _, err := r.client.CoreV1().Secrets(r.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return "", fmt.Errorf("creating config secret: %w", err)
	}

	// Configs, catalogs, and state are passed via K8s Secret mounted at /secrets.
	// The coordinator reads them from the secrets dir — no base64 CLI args needed.
	orchestratorArgs := []string{
		"synclet", "_orchestrate",
		"--job-id", opts.JobID,
		"--connection-id", opts.ConnectionID,
		"--server-addr", r.config.ServerAddr,
		"--data-dir", dataDir,
		"--source-id", opts.SourceID,
		"--dest-id", opts.DestinationID,
		"--source-image", opts.SourceImage,
		"--dest-image", opts.DestImage,
		"--secrets-dir", "/secrets",
	}

	// Namespace/prefix rewriting for coordinator's message router.
	if opts.NamespaceDefinition != "" {
		orchestratorArgs = append(orchestratorArgs, "--namespace-definition", opts.NamespaceDefinition)
	}
	if opts.CustomNamespaceFormat != "" {
		orchestratorArgs = append(orchestratorArgs, "--custom-namespace-format", opts.CustomNamespaceFormat)
	}
	if opts.StreamPrefix != "" {
		orchestratorArgs = append(orchestratorArgs, "--stream-prefix", opts.StreamPrefix)
	}

	// Build source command with proper Airbyte CLI args.
	sourceCmd := "read --config /shared/source-config.json --catalog /shared/source-catalog.json"
	if opts.State != nil {
		sourceCmd += " --state /shared/source-state.json"
	}
	orchestratorArgs = append(orchestratorArgs, "--source-cmd", sourceCmd)
	orchestratorArgs = append(orchestratorArgs, "--dest-cmd", "write --config /shared/dest-config.json --catalog /shared/dest-catalog.json")

	// Shared volume.
	sharedVolume := corev1.Volume{
		Name: "shared",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: resource.NewQuantity(1*1024*1024*1024, resource.BinarySI), // 1Gi
			},
		},
	}
	sharedMount := corev1.VolumeMount{
		Name:      "shared",
		MountPath: dataDir,
	}

	// Secret volume for config data (mounted read-only).
	secretVolume := corev1.Volume{
		Name: "sync-configs",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Optional:   boolPtr(false),
			},
		},
	}
	secretMount := corev1.VolumeMount{
		Name:      "sync-configs",
		MountPath: "/secrets",
		ReadOnly:  true,
	}

	// Orchestrator container (fully locked down).
	orchestratorContainer := corev1.Container{
		Name:         "orchestrator",
		Image:        r.config.SyncletImage,
		Command:      orchestratorArgs,
		VolumeMounts: []corev1.VolumeMount{sharedMount, secretMount},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			RunAsUser:                int64Ptr(1000),
			AllowPrivilegeEscalation: boolPtr(false),
			ReadOnlyRootFilesystem:   boolPtr(true),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
	}

	// Per-container resource requirements.
	sourceResources := buildResourceRequirements(opts.SourceMemoryLimit, opts.SourceCPULimit, opts.SourceMemoryRequest, opts.SourceCPURequest)
	destResources := buildResourceRequirements(opts.DestMemoryLimit, opts.DestCPULimit, opts.DestMemoryRequest, opts.DestCPURequest)

	// SubPath mounts: source container gets config/catalog/state directly from Secret.
	sourceVolumeMounts := []corev1.VolumeMount{sharedMount}
	sourceVolumeMounts = append(sourceVolumeMounts, corev1.VolumeMount{
		Name: "sync-configs", MountPath: "/shared/source-config.json", SubPath: "source-config", ReadOnly: true,
	})
	if opts.SourceCatalog != nil {
		sourceVolumeMounts = append(sourceVolumeMounts, corev1.VolumeMount{
			Name: "sync-configs", MountPath: "/shared/source-catalog.json", SubPath: "source-catalog", ReadOnly: true,
		})
	}
	if opts.State != nil {
		sourceVolumeMounts = append(sourceVolumeMounts, corev1.VolumeMount{
			Name: "sync-configs", MountPath: "/shared/source-state.json", SubPath: "source-state", ReadOnly: true,
		})
	}

	// SubPath mounts: destination container gets config/catalog directly from Secret.
	destVolumeMounts := []corev1.VolumeMount{sharedMount}
	destVolumeMounts = append(destVolumeMounts, corev1.VolumeMount{
		Name: "sync-configs", MountPath: "/shared/dest-config.json", SubPath: "dest-config", ReadOnly: true,
	})
	if opts.DestCatalog != nil {
		destVolumeMounts = append(destVolumeMounts, corev1.VolumeMount{
			Name: "sync-configs", MountPath: "/shared/dest-catalog.json", SubPath: "dest-catalog", ReadOnly: true,
		})
	}

	// Source container: waits for .ready, then runs source-run.sh.
	sourceContainer := corev1.Container{
		Name:         "source",
		Image:        opts.SourceImage,
		Command:      []string{"sh", "-c", fmt.Sprintf("while [ ! -f %s/.ready ]; do sleep 0.1; done; sh %s/source-run.sh", dataDir, dataDir)},
		VolumeMounts: sourceVolumeMounts,
		Resources:    sourceResources,
		// PullAlways for connector images to prevent stale tag reuse (SEC-13).
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
	}

	// Destination container: waits for .ready, then runs dest-run.sh.
	destContainer := corev1.Container{
		Name:         "destination",
		Image:        opts.DestImage,
		Command:      []string{"sh", "-c", fmt.Sprintf("while [ ! -f %s/.ready ]; do sleep 0.1; done; sh %s/dest-run.sh", dataDir, dataDir)},
		VolumeMounts: destVolumeMounts,
		Resources:    destResources,
		// PullAlways for connector images to prevent stale tag reuse (SEC-13).
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
	}

	var backoffLimit int32 = 0
	ttl := int32(600)
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers:    []corev1.Container{orchestratorContainer, sourceContainer, destContainer},
		Volumes:       []corev1.Volume{sharedVolume, secretVolume},
	}

	if r.config.ImagePullSecret != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: r.config.ImagePullSecret},
		}
	}

	// Apply pod-level scheduling from runtime config.
	if len(opts.Tolerations) > 0 {
		podSpec.Tolerations = opts.Tolerations
	}
	if len(opts.NodeSelector) > 0 {
		podSpec.NodeSelector = opts.NodeSelector
	}
	if opts.Affinity != nil {
		podSpec.Affinity = opts.Affinity
	}
	if opts.ServiceAccountName != "" {
		podSpec.ServiceAccountName = opts.ServiceAccountName
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"app":                   "synclet",
				"synclet.io/job":        jobName,
				"synclet.io/sync-job":   opts.JobID,
				"synclet.io/connection": sanitizeLabel(opts.ConnectionID),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":            "synclet",
						"synclet.io/job": jobName,
					},
				},
				Spec: podSpec,
			},
		},
	}

	created, err := r.client.BatchV1().Jobs(r.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		// Clean up the Secret since the Job won't exist to use it.
		// Use context.WithoutCancel so cleanup succeeds even if ctx was cancelled.
		_ = r.client.CoreV1().Secrets(r.namespace).Delete(context.WithoutCancel(ctx), secretName, metav1.DeleteOptions{})
		return "", fmt.Errorf("creating k8s job: %w", err)
	}

	return created.Name, nil
}

// ConnectorTaskOptions contains parameters to launch a connector task K8s Job.
type ConnectorTaskOptions struct {
	TaskID         string
	TaskType       string // "Check", "Spec", "Discover"
	Image          string // Connector image
	Config         []byte // Decrypted config JSON (nil for spec)
	InternalAPIURL string // Server address for gRPC callbacks
}

// LaunchConnectorTask creates a K8s Job with a 2-container pod (coordinator + connector)
// for executing connector tasks (check/spec/discover). Returns the K8s Job name.
// Simpler than LaunchSync: only 2 containers, shorter TTL, no FIFOs needed.
func (r *SyncRunner) LaunchConnectorTask(ctx context.Context, opts ConnectorTaskOptions) (string, error) {
	ts := time.Now().UnixNano()
	jobName := sanitizeK8sName(fmt.Sprintf("synclet-task-%s-%d", strings.ToLower(opts.TaskType), ts))
	dataDir := "/shared"

	// Create a K8s Secret for task config if provided.
	var secretName string
	if opts.Config != nil {
		secretName = sanitizeK8sName(fmt.Sprintf("synclet-task-%s-%d-secret", strings.ToLower(opts.TaskType), ts))
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: r.namespace,
				Labels: map[string]string{
					"app":             "synclet",
					"synclet.io/task": opts.TaskID,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"task-config": opts.Config,
			},
		}
		if _, err := r.client.CoreV1().Secrets(r.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("creating task config secret: %w", err)
		}
	}

	// Build coordinator args for task mode.
	// Note: --secrets-dir is not passed in task mode. The connector container
	// receives config.json directly via subPath mount from the Secret.
	orchestratorArgs := []string{
		"synclet", "_orchestrate",
		"--task-mode",
		"--task-id", opts.TaskID,
		"--task-type", opts.TaskType,
		"--server-addr", r.config.ServerAddr,
		"--data-dir", dataDir,
	}

	// Shared volume (256Mi, smaller than sync pods).
	sharedVolume := corev1.Volume{
		Name: "shared",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: resource.NewQuantity(256*1024*1024, resource.BinarySI),
			},
		},
	}
	sharedMount := corev1.VolumeMount{Name: "shared", MountPath: dataDir}

	// Secret volume for task config (only when config is provided).
	var taskSecretVolumes []corev1.Volume
	var connectorConfigMounts []corev1.VolumeMount
	if secretName != "" {
		taskSecretVolumes = append(taskSecretVolumes, corev1.Volume{
			Name: "task-configs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Optional:   boolPtr(false),
				},
			},
		})
		// SubPath mount: connector gets config.json directly from Secret.
		connectorConfigMounts = append(connectorConfigMounts, corev1.VolumeMount{
			Name:      "task-configs",
			MountPath: "/shared/config.json",
			SubPath:   "task-config",
			ReadOnly:  true,
		})
	}

	// Coordinator container (task mode). No secrets mount needed — the connector
	// container receives config.json directly via subPath.
	coordinatorContainer := corev1.Container{
		Name:         "coordinator",
		Image:        r.config.SyncletImage,
		Command:      orchestratorArgs,
		VolumeMounts: []corev1.VolumeMount{sharedMount},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			RunAsUser:                int64Ptr(1000),
			AllowPrivilegeEscalation: boolPtr(false),
			ReadOnlyRootFilesystem:   boolPtr(true),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
	}

	// Connector container: waits for .ready, then runs the coordinator-written run script.
	// The run script uses $AIRBYTE_ENTRYPOINT (set by Airbyte connector images) to invoke
	// the original entrypoint with the task args (check/spec/discover).
	connectorContainer := corev1.Container{
		Name:  "connector",
		Image: opts.Image,
		Command: []string{"sh", "-c", fmt.Sprintf(
			"while [ ! -f %s/.ready ]; do sleep 0.1; done; sh %s/connector-run.sh",
			dataDir, dataDir,
		)},
		VolumeMounts:    append([]corev1.VolumeMount{sharedMount}, connectorConfigMounts...),
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
	}

	var backoffLimit int32 = 0
	ttl := int32(120) // Shorter TTL for connector tasks (2 min after completion).
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers:    []corev1.Container{coordinatorContainer, connectorContainer},
		Volumes:       append([]corev1.Volume{sharedVolume}, taskSecretVolumes...),
	}

	if r.config.ImagePullSecret != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: r.config.ImagePullSecret},
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"app":                  "synclet",
				"synclet.io/job":       jobName,
				"synclet.io/task":      opts.TaskID,
				"synclet.io/task-type": strings.ToLower(opts.TaskType),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":            "synclet",
						"synclet.io/job": jobName,
					},
				},
				Spec: podSpec,
			},
		},
	}

	created, err := r.client.BatchV1().Jobs(r.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		if secretName != "" {
			// Clean up the Secret since the Job won't exist to use it.
			// Use context.WithoutCancel so cleanup succeeds even if ctx was cancelled.
			_ = r.client.CoreV1().Secrets(r.namespace).Delete(context.WithoutCancel(ctx), secretName, metav1.DeleteOptions{})
		}
		return "", fmt.Errorf("creating k8s connector task job: %w", err)
	}

	return created.Name, nil
}

// StopSync deletes a K8s Job and its pods.
func (r *SyncRunner) StopSync(ctx context.Context, jobName string) error {
	propagation := metav1.DeletePropagationForeground
	return r.client.BatchV1().Jobs(r.namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

func sanitizeK8sName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "_", "-")
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
}

func sanitizeLabel(value string) string {
	value = strings.ReplaceAll(value, "/", "_")
	if len(value) > 63 {
		value = value[:63]
	}
	return value
}

// buildResourceRequirements constructs K8s resource requirements from numeric limits and requests.
func buildResourceRequirements(memoryLimit int64, cpuLimit float64, memoryRequest int64, cpuRequest float64) corev1.ResourceRequirements {
	reqs := corev1.ResourceRequirements{}

	if memoryLimit > 0 || cpuLimit > 0 {
		limits := corev1.ResourceList{}
		if memoryLimit > 0 {
			limits[corev1.ResourceMemory] = *resource.NewQuantity(memoryLimit, resource.BinarySI)
		}
		if cpuLimit > 0 {
			limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(cpuLimit*1000), resource.DecimalSI)
		}
		reqs.Limits = limits
	}

	if memoryRequest > 0 || cpuRequest > 0 {
		requests := corev1.ResourceList{}
		if memoryRequest > 0 {
			requests[corev1.ResourceMemory] = *resource.NewQuantity(memoryRequest, resource.BinarySI)
		}
		if cpuRequest > 0 {
			requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(cpuRequest*1000), resource.DecimalSI)
		}
		reqs.Requests = requests
	}

	return reqs
}
