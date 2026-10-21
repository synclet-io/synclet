package helm_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// routeDoc is the part of an Ingress, HTTPRoute, Service or Deployment the MCP
// tests read.
type routeDoc struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Rules []struct {
			Host string `yaml:"host"`
			HTTP struct {
				Paths []struct {
					Path    string `yaml:"path"`
					Backend struct {
						Service struct {
							Name string `yaml:"name"`
							Port struct {
								Number int `yaml:"number"`
							} `yaml:"port"`
						} `yaml:"service"`
					} `yaml:"backend"`
				} `yaml:"paths"`
			} `yaml:"http"`
			Matches []struct {
				Path struct {
					Value string `yaml:"value"`
				} `yaml:"path"`
			} `yaml:"matches"`
			BackendRefs []struct {
				Name string `yaml:"name"`
				Port int    `yaml:"port"`
			} `yaml:"backendRefs"`
		} `yaml:"rules"`
		Ports []struct {
			Name       string `yaml:"name"`
			Port       int    `yaml:"port"`
			TargetPort any    `yaml:"targetPort"`
		} `yaml:"ports"`
		Template struct {
			Spec struct {
				Containers []struct {
					Name  string `yaml:"name"`
					Ports []struct {
						Name          string `yaml:"name"`
						ContainerPort int    `yaml:"containerPort"`
					} `yaml:"ports"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

func renderRoutes(t *testing.T, args ...string) map[string]routeDoc {
	t.Helper()

	docs := map[string]routeDoc{}
	dec := yaml.NewDecoder(bytes.NewReader(renderRaw(t, args...)))

	for {
		var doc routeDoc

		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return docs
		}

		require.NoError(t, err)

		if doc.Kind != "" {
			docs[doc.Kind+"/"+doc.Metadata.Name] = doc
		}
	}
}

var mcpPaths = []string{
	"/mcp", "/.well-known/oauth-protected-resource", "/.well-known/oauth-authorization-server", "/oauth/register", "/oauth/token",
}

// ingressPaths maps each path of an Ingress to its backend Service.
func ingressPaths(doc routeDoc) map[string]string {
	out := map[string]string{}

	for _, rule := range doc.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			out[path.Path] = path.Backend.Service.Name
		}
	}

	return out
}

func containerPorts(doc routeDoc) map[string]int {
	out := map[string]int{}

	for _, c := range doc.Spec.Template.Spec.Containers {
		for _, p := range c.Ports {
			out[p.Name] = p.ContainerPort
		}
	}

	return out
}

func TestChart_MCPDisabled(t *testing.T) {
	docs := renderRoutes(t)

	assert.NotContains(t, docs, "Service/t-synclet-mcp")
	assert.NotContains(t, docs, "Ingress/t-synclet-mcp")
	assert.Equal(t, map[string]string{"/": "t-synclet"}, ingressPaths(docs["Ingress/t-synclet"]))
	assert.NotContains(t, containerPorts(docs["Deployment/t-synclet-server"]), "mcp")

	cm := renderChart(t).get(t, "ConfigMap", "t-synclet")
	assert.NotContains(t, cm.Data, "MCP_ENABLED")
}

func TestChart_MCPShared(t *testing.T) {
	args := []string{"--set", "mcp.enabled=true"}
	docs := renderRoutes(t, args...)

	cm := renderChart(t, args...).get(t, "ConfigMap", "t-synclet")
	assert.Equal(t, "true", cm.Data["MCP_ENABLED"])
	assert.Equal(t, "true", cm.Data["MCP_WRITE_TOOLS"])
	assert.Equal(t, "0.0.0.0:8082", cm.Data["MCP_HTTP_SERVER_ADDR"])

	assert.Equal(t, 8082, containerPorts(docs["Deployment/t-synclet-server"])["mcp"])

	svc, ok := docs["Service/t-synclet-mcp"]
	require.True(t, ok)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, 80, svc.Spec.Ports[0].Port)
	assert.Equal(t, "mcp", svc.Spec.Ports[0].TargetPort)

	paths := ingressPaths(docs["Ingress/t-synclet"])
	for _, path := range mcpPaths {
		assert.Equal(t, "t-synclet-mcp", paths[path], path)
	}

	assert.Equal(t, "t-synclet", paths["/"])
	assert.NotContains(t, docs, "Ingress/t-synclet-mcp")

	assert.NotContains(t, cm.Data, "MCP_PUBLIC_URL")

	off := renderChart(t, "--set", "mcp.enabled=true", "--set", "mcp.writeTools=false").get(t, "ConfigMap", "t-synclet")
	assert.Equal(t, "false", off.Data["MCP_WRITE_TOOLS"])

	urls := renderChart(t, "--set", "mcp.enabled=true",
		"--set", "mcp.publicUrl=https://mcp.example.com", "--set", "mcp.uiUrl=https://synclet.internal").get(t, "ConfigMap", "t-synclet")
	assert.Equal(t, "https://mcp.example.com", urls.Data["MCP_PUBLIC_URL"])
	assert.Equal(t, "https://synclet.internal", urls.Data["MCP_UI_URL"])
}

func TestChart_MCPSharedGateway(t *testing.T) {
	docs := renderRoutes(t,
		"--set", "mcp.enabled=true",
		"--set", "server.ingress.enabled=false",
		"--set", "server.gateway.enabled=true",
		"--set", "server.gateway.parentRefs[0].name=gw",
		"--set", "server.gateway.hostnames[0]=synclet.example.com",
	)

	route := docs["HTTPRoute/t-synclet"]
	require.Len(t, route.Spec.Rules, 2)

	mcpRule := make([]string, 0, len(route.Spec.Rules[0].Matches))
	for _, match := range route.Spec.Rules[0].Matches {
		mcpRule = append(mcpRule, match.Path.Value)
	}

	assert.Equal(t, mcpPaths, mcpRule)
	assert.Equal(t, "t-synclet-mcp", route.Spec.Rules[0].BackendRefs[0].Name)
	assert.Equal(t, "t-synclet", route.Spec.Rules[1].BackendRefs[0].Name)
}

func TestChart_MCPSeparate(t *testing.T) {
	docs := renderRoutes(t,
		"--set", "mcp.enabled=true",
		"--set", "mcp.ingress.mode=separate",
		"--set", "mcp.ingress.hostname=mcp.example.com",
		"--set", "server.gateway.enabled=true",
		"--set", "server.gateway.parentRefs[0].name=gw",
		"--set", "server.gateway.hostnames[0]=synclet.example.com",
		"--set", "server.publicUrl=https://synclet.example.com",
	)

	assert.Equal(t, map[string]string{"/": "t-synclet"}, ingressPaths(docs["Ingress/t-synclet"]))
	assert.Len(t, docs["HTTPRoute/t-synclet"].Spec.Rules, 1)

	mcpIngress, ok := docs["Ingress/t-synclet-mcp"]
	require.True(t, ok)
	require.Len(t, mcpIngress.Spec.Rules, 1)
	assert.Equal(t, "mcp.example.com", mcpIngress.Spec.Rules[0].Host)

	paths := ingressPaths(mcpIngress)
	assert.Len(t, paths, len(mcpPaths))

	for _, path := range mcpPaths {
		assert.Equal(t, "t-synclet-mcp", paths[path], path)
	}
}

func TestChart_MCPSeparateNeedsHostname(t *testing.T) {
	out, err := helmTemplate(t, "--set", "mcp.enabled=true", "--set", "mcp.ingress.mode=separate").CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "mcp.ingress.hostname is required")

	out, err = helmTemplate(t, "--set", "mcp.enabled=true", "--set", "mcp.ingress.mode=public").CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "mcp.ingress.mode must be shared or separate")
}

func TestChart_MCPUIURLDefaultsToFrontend(t *testing.T) {
	cm := renderChart(t, "--set", "mcp.enabled=true").get(t, "ConfigMap", "t-synclet")
	assert.NotEmpty(t, cm.Data["MCP_UI_URL"])
	assert.Equal(t, cm.Data["WORKSPACE_FRONTEND_URL"], cm.Data["MCP_UI_URL"], "a password-only install advertises the frontend URL")

	cm = renderChart(t, "--set", "mcp.enabled=true", "--set", "frontend.url=https://app.example.com").get(t, "ConfigMap", "t-synclet")
	assert.Equal(t, "https://app.example.com", cm.Data["MCP_UI_URL"])

	cm = renderChart(t, "--set", "mcp.enabled=true", "--set", "server.publicUrl=https://synclet.example.com").get(t, "ConfigMap", "t-synclet")
	assert.Equal(t, "https://synclet.example.com", cm.Data["MCP_UI_URL"])
}

func TestChart_MCPUIURLOverReusedValues(t *testing.T) {
	versions := releasedValues(t)
	version := versions[len(versions)-1]

	docs := renderDocs(t, reuseValuesChart(t, version), "--set", "mcp.enabled=true,frontend.url=https://app.example.com")
	cm := configMapData(docs)["t-synclet"]
	require.NotNil(t, cm, "ConfigMap t-synclet over %s values", version)
	assert.Equal(t, "https://app.example.com", cm["MCP_UI_URL"], "over %s values", version)
	assert.Equal(t, "true", cm["MCP_WRITE_TOOLS"])
	assert.Equal(t, "0.0.0.0:8082", cm["MCP_HTTP_SERVER_ADDR"])
}

// reusedRoutes renders the chart over the newest released values, which have
// no mcp block, as helm upgrade --reuse-values does.
func reusedRoutes(t *testing.T, set string) map[string]routeDoc {
	t.Helper()

	versions := releasedValues(t)
	docs := map[string]routeDoc{}

	for _, raw := range renderDocs(t, reuseValuesChart(t, versions[len(versions)-1]), "--set", set) {
		out, err := yaml.Marshal(raw)
		require.NoError(t, err)

		var doc routeDoc
		require.NoError(t, yaml.Unmarshal(out, &doc))

		if doc.Kind != "" {
			docs[doc.Kind+"/"+doc.Metadata.Name] = doc
		}
	}

	return docs
}

func TestChart_MCPOverReusedValues(t *testing.T) {
	t.Run("shared", func(t *testing.T) {
		docs := reusedRoutes(t, "mcp.enabled=true")

		svc, ok := docs["Service/t-synclet-mcp"]
		require.True(t, ok)
		require.Len(t, svc.Spec.Ports, 1)
		assert.Equal(t, 80, svc.Spec.Ports[0].Port)
		assert.Equal(t, 8082, containerPorts(docs["Deployment/t-synclet-server"])["mcp"])

		paths := ingressPaths(docs["Ingress/t-synclet"])
		for _, path := range mcpPaths {
			assert.Equal(t, "t-synclet-mcp", paths[path], path)
		}

		assert.NotContains(t, docs, "Ingress/t-synclet-mcp")
	})

	t.Run("shared gateway", func(t *testing.T) {
		docs := reusedRoutes(t, "mcp.enabled=true,server.ingress.enabled=false,server.gateway.enabled=true,"+
			"server.gateway.parentRefs[0].name=gw,server.gateway.hostnames[0]=synclet.example.com")

		route := docs["HTTPRoute/t-synclet"]
		require.Len(t, route.Spec.Rules, 2)
		assert.Len(t, route.Spec.Rules[0].Matches, len(mcpPaths))
		require.Len(t, route.Spec.Rules[0].BackendRefs, 1)
		assert.Equal(t, 80, route.Spec.Rules[0].BackendRefs[0].Port)
	})

	t.Run("separate", func(t *testing.T) {
		docs := reusedRoutes(t, "mcp.enabled=true,mcp.ingress.mode=separate,mcp.ingress.hostname=mcp.example.com")

		assert.Equal(t, map[string]string{"/": "t-synclet"}, ingressPaths(docs["Ingress/t-synclet"]))

		mcpIngress, ok := docs["Ingress/t-synclet-mcp"]
		require.True(t, ok)
		require.Len(t, mcpIngress.Spec.Rules, 1)
		assert.Equal(t, "mcp.example.com", mcpIngress.Spec.Rules[0].Host)
		assert.Len(t, ingressPaths(mcpIngress), len(mcpPaths))
	})
}
