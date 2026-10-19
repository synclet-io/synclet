package helm_test

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type manifest struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name        string            `yaml:"name"`
		Namespace   string            `yaml:"namespace"`
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
	Data    map[string]string `yaml:"data"`
	Rules   []rbacRule        `yaml:"rules"`
	RoleRef struct {
		Kind string `yaml:"kind"`
		Name string `yaml:"name"`
	} `yaml:"roleRef"`
	Subjects []struct {
		Kind string `yaml:"kind"`
		Name string `yaml:"name"`
	} `yaml:"subjects"`
	Spec struct {
		ServiceName         string `yaml:"serviceName"`
		PodManagementPolicy string `yaml:"podManagementPolicy"`
		ClusterIP           string `yaml:"clusterIP"`
		Template            struct {
			Spec struct {
				ServiceAccountName string      `yaml:"serviceAccountName"`
				InitContainers     []container `yaml:"initContainers"`
				Containers         []container `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

type rbacRule struct {
	APIGroups       []string `yaml:"apiGroups"`
	Resources       []string `yaml:"resources"`
	Verbs           []string `yaml:"verbs"`
	NonResourceURLs []string `yaml:"nonResourceURLs"`
}

type container struct {
	Name string `yaml:"name"`
	Env  []struct {
		Name string `yaml:"name"`
	} `yaml:"env"`
	EnvFrom []struct {
		ConfigMapRef *struct {
			Name string `yaml:"name"`
		} `yaml:"configMapRef"`
	} `yaml:"envFrom"`
	StartupProbe   *probe `yaml:"startupProbe"`
	LivenessProbe  *probe `yaml:"livenessProbe"`
	ReadinessProbe *probe `yaml:"readinessProbe"`
}

type probe struct {
	HTTPGet struct {
		Path string `yaml:"path"`
		Port any    `yaml:"port"`
	} `yaml:"httpGet"`
}

type rendered []manifest

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

func helmTemplate(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	_, err := exec.LookPath("helm")
	require.NoError(t, err, "helm binary not found: the chart tests need helm on PATH")

	return exec.CommandContext(t.Context(), "helm", append([]string{"template", "t", "synclet", "--namespace", "synclet"}, args...)...) //nolint:gosec // arguments are fixed by the tests
}

func renderRaw(t *testing.T, args ...string) []byte {
	t.Helper()

	cmd := helmTemplate(t, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	require.NoError(t, err, stderr.String())

	return out
}

func renderChart(t *testing.T, args ...string) rendered {
	t.Helper()

	out := renderRaw(t, args...)

	var docs rendered

	dec := yaml.NewDecoder(bytes.NewReader(out))

	for {
		var m manifest

		err := dec.Decode(&m)
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		if m.Kind != "" {
			docs = append(docs, m)
		}
	}

	return docs
}

func (r rendered) names(kind string) []string {
	var names []string

	for _, m := range r {
		if m.Kind == kind {
			names = append(names, m.Metadata.Name)
		}
	}

	return names
}

func (r rendered) get(t *testing.T, kind, name string) manifest {
	t.Helper()

	for _, m := range r {
		if m.Kind == kind && m.Metadata.Name == name {
			return m
		}
	}

	require.Failf(t, "manifest not rendered", "%s %s", kind, name)

	return manifest{}
}

func (r rendered) bindingsOf(account string) []string {
	var roles []string

	for _, m := range r {
		if m.Kind != "RoleBinding" && m.Kind != "ClusterRoleBinding" {
			continue
		}

		for _, s := range m.Subjects {
			if s.Kind == "ServiceAccount" && s.Name == account {
				roles = append(roles, m.RoleRef.Name)
			}
		}
	}

	return roles
}

func TestChart_DefaultServiceAccounts(t *testing.T) {
	docs := renderChart(t)

	assert.Equal(t, []string{"t-synclet"}, docs.names("ServiceAccount"))
	assert.Equal(t, []string{"t-synclet-connector-runner"}, docs.bindingsOf("t-synclet"))
	assert.NotContains(t, docs.get(t, "ConfigMap", "t-synclet").Data, "PIPELINE_RUNTIME_SERVICE_ACCOUNT_NAME")
}

func TestChart_DistributedExecutorServiceAccount(t *testing.T) {
	docs := renderChart(t,
		"--set", "mode=distributed",
		"--set", "executor.serviceAccount.annotations.iam\\.gke\\.io/gcp-service-account=executor@p.iam.gserviceaccount.com",
	)

	assert.ElementsMatch(t, []string{"t-synclet", "t-synclet-executor"}, docs.names("ServiceAccount"))
	assert.Equal(t, "t-synclet-executor", docs.get(t, "StatefulSet", "t-synclet-executor").Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, "t-synclet", docs.get(t, "Deployment", "t-synclet-server").Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, []string{"t-synclet-connector-runner"}, docs.bindingsOf("t-synclet-executor"))
	assert.Equal(t,
		map[string]string{"iam.gke.io/gcp-service-account": "executor@p.iam.gserviceaccount.com"},
		docs.get(t, "ServiceAccount", "t-synclet-executor").Metadata.Annotations,
	)
}

func TestChart_ExecutorExistingServiceAccount(t *testing.T) {
	docs := renderChart(t,
		"--set", "mode=distributed",
		"--set", "executor.serviceAccount.create=false",
		"--set", "executor.serviceAccount.name=byo-executor",
	)

	assert.Equal(t, []string{"t-synclet"}, docs.names("ServiceAccount"))
	assert.Equal(t, "byo-executor", docs.get(t, "StatefulSet", "t-synclet-executor").Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, []string{"t-synclet-connector-runner"}, docs.bindingsOf("byo-executor"))
}

func TestChart_ConnectorServiceAccount(t *testing.T) {
	docs := renderChart(t,
		"--set", "k8s.namespace=connectors",
		"--set", "pipeline.serviceAccount.create=true",
		"--set", "pipeline.serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn=arn:aws:iam::1:role/connectors",
	)

	sa := docs.get(t, "ServiceAccount", "t-synclet-connector")
	assert.Equal(t, "connectors", sa.Metadata.Namespace)
	assert.Equal(t, map[string]string{"eks.amazonaws.com/role-arn": "arn:aws:iam::1:role/connectors"}, sa.Metadata.Annotations)
	assert.Empty(t, docs.bindingsOf("t-synclet-connector"))
	assert.Equal(t, "t-synclet-connector", docs.get(t, "ConfigMap", "t-synclet").Data["PIPELINE_RUNTIME_SERVICE_ACCOUNT_NAME"])
}

// namespacedResources lists every resource, by API group, the chart's Role may
// grant; each is namespaced, so the Role cannot reach past its namespace.
var namespacedResources = map[string][]string{
	"":      {"pods", "pods/log", "configmaps", "secrets", "events"},
	"batch": {"jobs"},
}

func TestChart_RBACStaysInNamespace(t *testing.T) {
	modes := map[string][]string{
		"standalone":                      nil,
		"standalone with connector SA":    {"--set", "pipeline.serviceAccount.create=true"},
		"distributed":                     {"--set", "mode=distributed"},
		"distributed with connector SA":   {"--set", "mode=distributed", "--set", "pipeline.serviceAccount.create=true", "--set", "k8s.namespace=connectors"},
		"distributed with existing SAs":   {"--set", "mode=distributed", "--set", "executor.serviceAccount.create=false", "--set", "executor.serviceAccount.name=byo-executor"},
		"distributed without executor SA": {"--set", "mode=distributed", "--set", "serviceAccount.create=false", "--set", "executor.serviceAccount.create=false"},
	}

	for name, args := range modes {
		t.Run(name, func(t *testing.T) {
			docs := renderChart(t, args...)

			assert.Empty(t, docs.names("ClusterRole"))
			assert.Empty(t, docs.names("ClusterRoleBinding"))

			for _, m := range docs {
				switch m.Kind {
				case "Role":
					for _, rule := range m.Rules {
						assert.Empty(t, rule.NonResourceURLs, "Role %s", m.Metadata.Name)

						for _, group := range rule.APIGroups {
							allowed, known := namespacedResources[group]
							require.True(t, known, "Role %s grants API group %q", m.Metadata.Name, group)

							for _, resource := range rule.Resources {
								assert.Contains(t, allowed, resource, "Role %s grants %s in API group %q", m.Metadata.Name, resource, group)
							}
						}
					}
				case "RoleBinding":
					assert.Equal(t, "Role", m.RoleRef.Kind, "RoleBinding %s", m.Metadata.Name)
				}
			}

			if contains(args, "pipeline.serviceAccount.create=true") {
				docs.get(t, "ServiceAccount", "t-synclet-connector")
				assert.Empty(t, docs.bindingsOf("t-synclet-connector"))
			}
		})
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}

	return false
}

func TestChart_DockerOwnerID(t *testing.T) {
	_, set := renderChart(t).get(t, "ConfigMap", "t-synclet").Data["DOCKER_EXECUTOR_OWNER_ID"]
	assert.False(t, set)

	cm := renderChart(t, "--set", "pipeline.dockerOwnerId=blue").get(t, "ConfigMap", "t-synclet")
	assert.Equal(t, "blue", cm.Data["DOCKER_EXECUTOR_OWNER_ID"])
}

func (m manifest) container(t *testing.T, name string) container {
	t.Helper()

	for _, c := range m.Spec.Template.Spec.Containers {
		if c.Name == name {
			return c
		}
	}

	require.Failf(t, "container not rendered", "%s", name)

	return container{}
}

func TestChart_ServerProbeDefaults(t *testing.T) {
	server := renderChart(t).get(t, "Deployment", "t-synclet-server").container(t, "synclet")

	require.NotNil(t, server.LivenessProbe)
	require.NotNil(t, server.ReadinessProbe)
	assert.Equal(t, "/live", server.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, "internal", server.LivenessProbe.HTTPGet.Port)
	assert.Equal(t, "/ready", server.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, "internal", server.ReadinessProbe.HTTPGet.Port)
}

func TestChart_ServerProbeOverrides(t *testing.T) {
	server := renderChart(t,
		"--set", "server.livenessProbe.path=/health",
		"--set", "server.livenessProbe.port=8080",
		"--set", "server.readinessProbe.path=/health",
		"--set", "server.readinessProbe.port=public",
	).get(t, "Deployment", "t-synclet-server").container(t, "synclet")

	require.NotNil(t, server.LivenessProbe)
	require.NotNil(t, server.ReadinessProbe)
	assert.Equal(t, "/health", server.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, 8080, server.LivenessProbe.HTTPGet.Port)
	assert.Equal(t, "/health", server.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, "public", server.ReadinessProbe.HTTPGet.Port)
}

func TestChart_DistributedPodsWaitForDatabase(t *testing.T) {
	inits := func(r rendered, name string) []string {
		return initNames(r.get(t, "Deployment", name))
	}

	r := renderChart(t, "--set", "mode=distributed")
	assert.Equal(t, []string{"wait-postgres", "migrate"}, inits(r, "t-synclet-server"))
	assert.Equal(t, []string{"wait-postgres", "migrate"}, inits(r, "t-synclet-jobs"))

	r = renderChart(t, "--set", "mode=distributed", "--set", "postgresql.enabled=false",
		"--set", "externalDatabase.host=db", "--set", "externalDatabase.password=x")
	assert.Equal(t, []string{"migrate"}, inits(r, "t-synclet-server"))
	assert.Equal(t, []string{"migrate"}, inits(r, "t-synclet-jobs"))
}

func initNames(m manifest) []string {
	containers := m.Spec.Template.Spec.InitContainers

	names := make([]string, 0, len(containers))
	for _, c := range containers {
		names = append(names, c.Name)
	}

	return names
}

func TestChart_ExecutorGetsOnlyExecutorSecrets(t *testing.T) {
	envNames := func(c container) []string {
		names := make([]string, 0, len(c.Env))
		for _, e := range c.Env {
			names = append(names, e.Name)
		}

		return names
	}

	r := renderChart(t, "--set", "mode=distributed")

	assert.Equal(t, []string{"EXECUTOR_API_TOKEN"}, envNames(r.get(t, "StatefulSet", "t-synclet-executor").container(t, "executor")))

	server := envNames(r.get(t, "Deployment", "t-synclet-server").container(t, "synclet"))
	assert.Contains(t, server, "DB_DSN")
	assert.Contains(t, server, "PIPELINE_EXECUTOR_API_TOKEN")
	assert.NotContains(t, server, "EXECUTOR_API_TOKEN")
}

func TestChart_ExecutorStatefulSet(t *testing.T) {
	r := renderChart(t, "--set", "mode=distributed")

	assert.NotContains(t, r.names("Deployment"), "t-synclet-executor")

	executor := r.get(t, "StatefulSet", "t-synclet-executor")
	assert.Equal(t, "t-synclet-executor", executor.Spec.ServiceName)
	assert.Equal(t, "Parallel", executor.Spec.PodManagementPolicy)
	assert.Empty(t, initNames(executor), "the executor reaches the control plane through its API, not Postgres")

	assert.Equal(t, "None", r.get(t, "Service", "t-synclet-executor").Spec.ClusterIP)

	var configMaps []string

	for _, from := range executor.container(t, "executor").EnvFrom {
		if from.ConfigMapRef != nil {
			configMaps = append(configMaps, from.ConfigMapRef.Name)
		}
	}

	assert.Equal(t, []string{"t-synclet-executor"}, configMaps)

	cm := r.get(t, "ConfigMap", "t-synclet-executor").Data
	assert.Equal(t, "http://t-synclet-internal.synclet.svc.cluster.local:8081", cm["EXECUTOR_API_URL"])
	assert.Equal(t, "http://t-synclet-internal.synclet.svc.cluster.local:8081", cm["K8S_EXECUTOR_SERVER_ADDR"])
	assert.Contains(t, cm, "K8S_EXECUTOR_SYNCLET_IMAGE")
	assert.Contains(t, cm, "PIPELINE_IDLE_TIMEOUT")

	for key := range cm {
		assert.NotRegexp(t, `^(DB_|AUTH_|WORKSPACE_|NOTIFY_|PUBLIC_HTTP_|RATE_LIMIT_|DOCKER_EXECUTOR_|PIPELINE_JOB_|PIPELINE_LOG_|PIPELINE_CONNECTOR_)`, key, "control-plane setting in the executor ConfigMap")
	}

	r = renderChart(t)
	assert.NotContains(t, r.names("StatefulSet"), "t-synclet-executor")
	assert.NotContains(t, r.names("ConfigMap"), "t-synclet-executor")
	assert.NotContains(t, r.names("Service"), "t-synclet-executor")
}

func TestChart_K8sOwnerID(t *testing.T) {
	_, set := renderChart(t, "--set", "mode=distributed").get(t, "ConfigMap", "t-synclet-executor").Data["K8S_EXECUTOR_OWNER_ID"]
	assert.False(t, set, "the executor defaults to its pod name")

	cm := renderChart(t, "--set", "mode=distributed", "--set", "k8s.ownerId=blue").get(t, "ConfigMap", "t-synclet-executor")
	assert.Equal(t, "blue", cm.Data["K8S_EXECUTOR_OWNER_ID"])

	assert.Equal(t, "t-synclet", renderChart(t).get(t, "ConfigMap", "t-synclet").Data["K8S_EXECUTOR_OWNER_ID"])
	assert.Equal(t, "blue", renderChart(t, "--set", "k8s.ownerId=blue").get(t, "ConfigMap", "t-synclet").Data["K8S_EXECUTOR_OWNER_ID"])
}

func TestChart_K8sLocalVolume(t *testing.T) {
	_, set := renderChart(t).get(t, "ConfigMap", "t-synclet").Data["K8S_EXECUTOR_LOCAL_VOLUME_CLAIM"]
	assert.False(t, set, "connector pods mount an emptyDir at /local by default")

	assert.Equal(t, "synclet-local",
		renderChart(t, "--set", "k8s.localVolume=synclet-local").get(t, "ConfigMap", "t-synclet").Data["K8S_EXECUTOR_LOCAL_VOLUME_CLAIM"])

	cm := renderChart(t, "--set", "mode=distributed", "--set", "k8s.localVolume=synclet-local").get(t, "ConfigMap", "t-synclet-executor")
	assert.Equal(t, "synclet-local", cm.Data["K8S_EXECUTOR_LOCAL_VOLUME_CLAIM"])
}

func TestChart_ExecutorStrategyRenamed(t *testing.T) {
	out, err := helmTemplate(t, "--set", "mode=distributed", "--set", "executor.strategy.rollingUpdate.maxSurge=1").CombinedOutput()
	require.Error(t, err, "render must fail while executor.strategy is set")
	assert.Contains(t, string(out), "executor.strategy was renamed to executor.updateStrategy")
}

func TestChart_ServerStartupProbe(t *testing.T) {
	server := renderChart(t).get(t, "Deployment", "t-synclet-server").container(t, "synclet")

	require.NotNil(t, server.StartupProbe)
	assert.Equal(t, "/live", server.StartupProbe.HTTPGet.Path)
	assert.Equal(t, "internal", server.StartupProbe.HTTPGet.Port)

	server = renderChart(t,
		"--set", "server.startupProbe.path=/health",
		"--set", "server.startupProbe.port=public",
	).get(t, "Deployment", "t-synclet-server").container(t, "synclet")

	require.NotNil(t, server.StartupProbe)
	assert.Equal(t, "/health", server.StartupProbe.HTTPGet.Path)
	assert.Equal(t, "public", server.StartupProbe.HTTPGet.Port)

	server = renderChart(t, "--set", "server.startupProbe.port=8081").get(t, "Deployment", "t-synclet-server").container(t, "synclet")

	require.NotNil(t, server.StartupProbe)
	assert.Equal(t, 8081, server.StartupProbe.HTTPGet.Port)
}

func TestChart_GatewayParentRefNameRequired(t *testing.T) {
	out, err := helmTemplate(t,
		"--set", "server.ingress.enabled=false",
		"--set", "server.gateway.enabled=true",
		"--set", "server.gateway.parentRefs[0].namespace=gateway-system",
		"--set", "server.gateway.hostnames[0]=synclet.example.com",
	).CombinedOutput()
	require.Error(t, err, "render must fail while a parentRef has no name")
	assert.Contains(t, string(out), "server.gateway.parentRefs[].name is required")

	route := renderChart(t,
		"--set", "server.ingress.enabled=false",
		"--set", "server.gateway.enabled=true",
		"--set", "server.gateway.parentRefs[0].name=gw",
		"--set", "server.gateway.hostnames[0]=synclet.example.com",
	)
	assert.Equal(t, []string{"t-synclet"}, route.names("HTTPRoute"))
}

// goldenSecrets pins the values the chart otherwise generates at random, so the
// golden renders are stable.
var goldenSecrets = []string{
	"--set", "auth.jwtSecret=golden-jwt-secret",
	"--set", "encryption.key=golden-encryption-key",
	"--set", "internalApi.secret=golden-internal-api-secret",
	"--set", "postgresql.auth.password=golden-postgres-password",
}

// checksumAnnotation matches the pod template checksums, which hash rendered
// manifests that carry the chart version.
var checksumAnnotation = regexp.MustCompile(`(checksum/[a-z]+): [0-9a-f]{64}`)

// documentSeparator matches a YAML document separator line.
var documentSeparator = regexp.MustCompile(`(?m)^---[ \t]*$`)

// normalizeManifests rewrites a multi-document render canonically: every
// non-empty document, trimmed of surrounding whitespace, follows its own "---"
// line. Helm 3 and helm 4 differ in the blank lines around separators, so the
// golden files compare equal under both.
func normalizeManifests(manifests string) string {
	var out strings.Builder

	for _, doc := range documentSeparator.Split(manifests, -1) {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		out.WriteString("---\n")
		out.WriteString(doc)
		out.WriteString("\n")
	}

	return out.String()
}

func TestNormalizeManifests(t *testing.T) {
	helm3 := "---\n# Source: a.yaml\nkind: A\n---\n# Source: b.yaml\nkind: B\n"
	helm4 := "---\n# Source: a.yaml\nkind: A\n\n---\n# Source: b.yaml\nkind: B\n\n"

	assert.Equal(t, helm3, normalizeManifests(helm3))
	assert.Equal(t, helm3, normalizeManifests(helm4))
}

// TestChart_Golden compares whole renders with the files in testdata. The
// chart version, appVersion and checksum annotations are replaced with
// placeholders so release bumps leave the goldens untouched. Run with -update
// to rewrite them after an intended template change.
func TestChart_Golden(t *testing.T) {
	var chart struct {
		Version    string `yaml:"version"`
		AppVersion string `yaml:"appVersion"`
	}

	chartYAML, err := os.ReadFile(filepath.Join("synclet", "Chart.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(chartYAML, &chart))

	placeholders := strings.NewReplacer(
		"synclet-"+chart.Version, "synclet-CHART_VERSION",
		chart.AppVersion, "APP_VERSION",
	)

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "defaults"},
		{name: "distributed", args: []string{
			"--set", "mode=distributed",
			"--set", "executor.livenessProbe.enabled=true",
			"--set", "executor.readinessProbe.enabled=true",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := placeholders.Replace(string(renderRaw(t, append(slices.Clone(goldenSecrets), tc.args...)...)))
			got = normalizeManifests(checksumAnnotation.ReplaceAllString(got, "$1: SHA256"))

			golden := filepath.Join("testdata", tc.name+".golden.yaml")

			if *update {
				require.NoError(t, os.WriteFile(golden, []byte(got), 0o600))
			}

			want, err := os.ReadFile(golden) //nolint:gosec // path built from the fixed test case names
			require.NoError(t, err, "run go test ./deploy/helm/ -run TestChart_Golden -update to create it")
			assert.Equal(t, string(want), got, "render differs from %s; rerun with -update if the change is intended", golden)
		})
	}
}

const k8sExecutorDir = "../../modules/pipeline/pipelineexec/pipelineexeck8s"

// k8sResourceCall matches a typed client call such as
// client.CoreV1().Secrets(ns).Create(, capturing the resource accessor and the verb.
var k8sResourceCall = regexp.MustCompile(`\.(Jobs|Pods|Secrets|ConfigMaps|Events)\([^()]*\)(\.\w+\()?`)

// executorAPICalls returns the RBAC verbs the k8s executor code uses, keyed by
// "apiGroup/resource". Every resource accessor must be followed directly by its
// verb so that no call escapes the scan.
func executorAPICalls(t *testing.T) map[string][]string {
	t.Helper()

	groups := map[string]string{"Jobs": "batch/jobs", "Pods": "/pods", "Secrets": "/secrets", "ConfigMaps": "/configmaps", "Events": "/events"}
	used := map[string][]string{}

	files, err := filepath.Glob(filepath.Join(k8sExecutorDir, "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		src, err := os.ReadFile(file) //nolint:gosec // paths come from a glob over the executor package
		require.NoError(t, err)

		for _, m := range k8sResourceCall.FindAllStringSubmatch(string(src), -1) {
			require.NotEmpty(t, m[2], "%s: %s is not called directly; chain the verb so this test sees it", file, m[0])

			resource, verb := groups[m[1]], strings.ToLower(strings.Trim(m[2], ".("))
			if verb == "getlogs" {
				resource, verb = "/pods/log", "get"
			}

			if !slices.Contains(used[resource], verb) {
				used[resource] = append(used[resource], verb)
			}
		}
	}

	require.NotEmpty(t, used)

	return used
}

func TestChart_ExecutorRoleGrantsVerbsTheCodeUses(t *testing.T) {
	used := executorAPICalls(t)

	modes := map[string][]string{
		"standalone":  nil,
		"distributed": {"--set", "mode=distributed"},
	}

	for name, args := range modes {
		t.Run(name, func(t *testing.T) {
			role := renderChart(t, args...).get(t, "Role", "t-synclet-connector-runner")

			granted := map[string][]string{}

			for _, rule := range role.Rules {
				for _, group := range rule.APIGroups {
					for _, resource := range rule.Resources {
						granted[group+"/"+resource] = append(granted[group+"/"+resource], rule.Verbs...)
					}
				}
			}

			for resource, verbs := range used {
				for _, verb := range verbs {
					assert.Contains(t, granted[resource], verb, "Role grants no %q on %s", verb, resource)
				}
			}
		})
	}
}
