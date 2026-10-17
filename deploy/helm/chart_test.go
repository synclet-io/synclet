package helm_test

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
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
	RoleRef struct {
		Kind string `yaml:"kind"`
		Name string `yaml:"name"`
	} `yaml:"roleRef"`
	Subjects []struct {
		Kind string `yaml:"kind"`
		Name string `yaml:"name"`
	} `yaml:"subjects"`
	Spec struct {
		Template struct {
			Spec struct {
				ServiceAccountName string      `yaml:"serviceAccountName"`
				InitContainers     []container `yaml:"initContainers"`
				Containers         []container `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

type container struct {
	Name           string `yaml:"name"`
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

func renderChart(t *testing.T, args ...string) rendered {
	t.Helper()

	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm binary not found")
	}

	cmd := exec.CommandContext(t.Context(), "helm", append([]string{"template", "t", "synclet", "--namespace", "synclet"}, args...)...) //nolint:gosec // arguments are fixed by the tests

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	require.NoError(t, err, stderr.String())

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
	assert.Equal(t, "t-synclet-executor", docs.get(t, "Deployment", "t-synclet-executor").Spec.Template.Spec.ServiceAccountName)
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
	assert.Equal(t, "byo-executor", docs.get(t, "Deployment", "t-synclet-executor").Spec.Template.Spec.ServiceAccountName)
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
		containers := r.get(t, "Deployment", name).Spec.Template.Spec.InitContainers

		names := make([]string, 0, len(containers))
		for _, c := range containers {
			names = append(names, c.Name)
		}

		return names
	}

	r := renderChart(t, "--set", "mode=distributed")
	assert.Equal(t, []string{"wait-postgres", "migrate"}, inits(r, "t-synclet-server"))
	assert.Equal(t, []string{"wait-postgres", "migrate"}, inits(r, "t-synclet-jobs"))
	assert.Equal(t, []string{"wait-postgres"}, inits(r, "t-synclet-executor"))

	r = renderChart(t, "--set", "mode=distributed", "--set", "postgresql.enabled=false",
		"--set", "externalDatabase.host=db", "--set", "externalDatabase.password=x")
	assert.Equal(t, []string{"migrate"}, inits(r, "t-synclet-server"))
	assert.Equal(t, []string{"migrate"}, inits(r, "t-synclet-jobs"))
	assert.Empty(t, inits(r, "t-synclet-executor"))
}
