package helm_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// releasedValues are the values.yaml files of earlier chart releases, copied
// from their synclet-helm-v<version> tags. Add the values of each new release
// here so the next one is rendered over them.
var releasedValues = []string{"0.1.3", "0.2.0", "0.3.0"}

// reuseValuesChart copies the chart into a temp dir with values.yaml replaced
// by a released one: helm upgrade --reuse-values renders the new templates over
// the previous release's values, without the new chart's defaults.
func reuseValuesChart(t *testing.T, version string) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "synclet")

	require.NoError(t, os.CopyFS(dir, os.DirFS("synclet")))

	values, err := os.ReadFile(filepath.Join("testdata", "release-values", version+".yaml")) //nolint:gosec // version comes from releasedValues
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
// release, as helm upgrade --reuse-values does.
func TestChart_ReuseValues(t *testing.T) {
	for _, version := range releasedValues {
		for _, mode := range []string{"standalone", "distributed"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				docs := renderDocs(t, reuseValuesChart(t, version), "--set", "mode="+mode)
				defaults := configMapData(renderDocs(t, "synclet", "--set", "mode="+mode))

				for name, data := range configMapData(docs) {
					for key, want := range defaults[name] {
						if want == "" {
							continue
						}

						assert.NotEmpty(t, data[key], "ConfigMap %s key %s is empty over %s values (default %q)", name, key, version, want)
					}
				}

				for _, p := range probes(docs) {
					if p.httpGet == nil {
						continue
					}

					assert.NotEmpty(t, p.httpGet["port"], "%s/%s %s has no port", p.workload, p.container, p.kind)

					if p.container == "synclet" {
						assert.Contains(t, []any{"/live", "/ready"}, p.httpGet["path"], "%s/%s %s path", p.workload, p.container, p.kind)
					}
				}
			})
		}
	}
}
