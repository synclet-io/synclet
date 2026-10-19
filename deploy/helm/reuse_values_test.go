package helm_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// releasedValues lists the chart versions in testdata/release-values: the
// values.yaml of each release, copied from its synclet-helm-v<version> tag.
func releasedValues(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join("testdata", "release-values"))
	require.NoError(t, err)

	var versions []*semver.Version

	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".yaml"); ok {
			version, err := semver.StrictNewVersion(name)
			require.NoError(t, err, "testdata/release-values/%s is not named after a chart version", e.Name())

			versions = append(versions, version)
		}
	}

	slices.SortFunc(versions, func(a, b *semver.Version) int { return a.Compare(b) })

	names := make([]string, len(versions))
	for i, v := range versions {
		names[i] = v.String()
	}

	return names
}

// TestReleasedValues_CoverLastRelease fails until the values of the chart
// version release-please last released are copied to testdata/release-values.
func TestReleasedValues_CoverLastRelease(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".release-please-manifest.json"))
	require.NoError(t, err)

	var manifest map[string]string
	require.NoError(t, json.Unmarshal(raw, &manifest))

	released := manifest["deploy/helm/synclet"]
	require.NotEmpty(t, released, "release-please manifest has no deploy/helm/synclet version")

	versions := releasedValues(t)
	require.NotEmpty(t, versions)
	assert.Equal(t, released, versions[len(versions)-1],
		"copy `git show synclet-helm-v%s:deploy/helm/synclet/values.yaml` to testdata/release-values/%s.yaml", released, released)
}

// reuseValuesChart copies the chart into a temp dir with values.yaml replaced
// by a released one: helm upgrade --reuse-values renders the new templates over
// the previous release's values, without the new chart's defaults.
func reuseValuesChart(t *testing.T, version string) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "synclet")

	require.NoError(t, os.CopyFS(dir, os.DirFS("synclet")))

	values, err := os.ReadFile(filepath.Join("testdata", "release-values", version+".yaml")) //nolint:gosec // version comes from the testdata directory
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), values, 0o600)) //nolint:gosec // writes into the test temp dir

	return dir
}

func renderDocs(t *testing.T, chart string, args ...string) []map[string]any {
	t.Helper()

	_, err := exec.LookPath("helm")
	require.NoError(t, err, "helm binary not found: the chart tests need helm on PATH")

	cmd := exec.CommandContext(t.Context(), "helm", append([]string{"template", "t", chart, "--namespace", "synclet"}, args...)...) //nolint:gosec // arguments are fixed by the tests

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	require.NoError(t, err, stderr.String())

	var docs []map[string]any

	dec := yaml.NewDecoder(bytes.NewReader(out))

	for {
		var doc map[string]any

		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		if doc != nil {
			docs = append(docs, doc)
		}
	}

	return docs
}

func configMapData(docs []map[string]any) map[string]map[string]any {
	data := map[string]map[string]any{}

	for _, doc := range docs {
		if doc["kind"] != "ConfigMap" {
			continue
		}

		name, _ := doc["metadata"].(map[string]any)["name"].(string)
		entries, _ := doc["data"].(map[string]any)
		data[name] = entries
	}

	return data
}

type renderedProbe struct {
	workload, container, kind string
	httpGet                   map[string]any
}

func probes(docs []map[string]any) []renderedProbe {
	var out []renderedProbe

	for _, doc := range docs {
		spec, _ := doc["spec"].(map[string]any)
		template, _ := spec["template"].(map[string]any)
		podSpec, _ := template["spec"].(map[string]any)
		containers, _ := podSpec["containers"].([]any)
		workload, _ := doc["metadata"].(map[string]any)["name"].(string)

		for _, c := range containers {
			container, _ := c.(map[string]any)
			name, _ := container["name"].(string)

			for _, kind := range []string{"startupProbe", "readinessProbe", "livenessProbe"} {
				p, ok := container[kind].(map[string]any)
				if !ok {
					continue
				}

				httpGet, _ := p["httpGet"].(map[string]any)
				out = append(out, renderedProbe{workload: workload, container: name, kind: kind, httpGet: httpGet})
			}
		}
	}

	return out
}

// TestChart_ReuseValues renders the chart over the values of each earlier
// release, as helm upgrade --reuse-values does. Every ConfigMap key must render
// as it does with the new defaults under the old release's values: keys the
// old release set keep its value, the rest fall back to values.yaml.
func TestChart_ReuseValues(t *testing.T) {
	for _, version := range releasedValues(t) {
		for _, mode := range []string{"standalone", "distributed"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				set := "mode=" + mode + ",jobs.livenessProbe.enabled=true,jobs.readinessProbe.enabled=true," +
					"executor.livenessProbe.enabled=true,executor.readinessProbe.enabled=true"

				docs := renderDocs(t, reuseValuesChart(t, version), "--set", set)
				got := configMapData(docs)
				want := configMapData(renderDocs(t, "synclet", "--set", set, "-f", filepath.Join("testdata", "release-values", version+".yaml")))

				for name, data := range want {
					for key, value := range data {
						assert.Equal(t, value, got[name][key], "ConfigMap %s key %s over %s values", name, key, version)
					}
				}

				for _, p := range probes(docs) {
					if p.httpGet == nil {
						continue
					}

					assert.Contains(t, []any{"internal", 8081}, p.httpGet["port"], "%s/%s %s port", p.workload, p.container, p.kind)
					assert.Contains(t, []any{"/live", "/ready"}, p.httpGet["path"], "%s/%s %s path", p.workload, p.container, p.kind)
				}
			})
		}
	}
}
