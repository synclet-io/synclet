package app

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestConfigInventory fails when .env.dist, the README environment table, the
// docker-compose file or the Helm chart names a variable the binary never reads.
func TestConfigInventory(t *testing.T) {
	configs, err := configInventory()
	if err != nil {
		t.Fatalf("config inventory: %v", err)
	}

	known := map[string]bool{}

	for _, config := range configs {
		for _, field := range config.Fields {
			known[field.Key] = true
		}
	}

	// Read directly with os.Getenv rather than through a config struct.
	for _, key := range []string{"KUBERNETES_SERVICE_HOST"} {
		known[key] = true
	}

	oidcProviderKey := regexp.MustCompile(`^AUTH_OIDC_(<SLUG>|\{\{ \$slug \}\})_[A-Z_]+$`)

	sources := map[string]struct {
		path    string
		pattern *regexp.Regexp
		section [2]string
	}{
		".env.dist": {
			path:    "../.env.dist",
			pattern: regexp.MustCompile(`(?m)^(?:# )?([A-Z][A-Z0-9_]*)=`),
		},
		"README environment table": {
			path:    "../README.md",
			pattern: regexp.MustCompile("(?m)^\\| `([A-Z][A-Z0-9_<>]*\\*?)`"),
			section: [2]string{"### Environment Variables", "### Connector Configuration"},
		},
		"docker-compose.yml": {
			path:    "../docker-compose.yml",
			pattern: regexp.MustCompile(`(?m)^ {6}([A-Z][A-Z0-9_]*):`),
			section: [2]string{"services:", "\n  postgres:\n"},
		},
		"helm configmap": {
			path:    "../deploy/helm/synclet/templates/configmap.yaml",
			pattern: regexp.MustCompile(`(?m)^ {2}([A-Z][A-Z0-9_]*(?:\{\{ \$slug \}\}[A-Z0-9_]*)?):`),
		},
		"helm secret env": {
			path:    "../deploy/helm/synclet/templates/_helpers.tpl",
			pattern: regexp.MustCompile(`(?m)^- name: ([A-Z][A-Z0-9_]*)$`),
		},
	}

	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(source.path)
			if err != nil {
				t.Fatalf("read %s: %v", source.path, err)
			}

			text := string(content)
			if source.section[0] != "" {
				start := strings.Index(text, source.section[0])
				end := strings.Index(text, source.section[1])

				if start < 0 || end < start {
					t.Fatalf("section %q not found in %s", source.section[0], source.path)
				}

				text = text[start:end]
			}

			matches := source.pattern.FindAllStringSubmatch(text, -1)
			if len(matches) == 0 {
				t.Fatalf("no variables found in %s", source.path)
			}

			for _, match := range matches {
				key := strings.TrimSuffix(match[1], "*")
				if known[key] || oidcProviderKey.MatchString(key) || strings.HasSuffix(match[1], "*") && hasKnownPrefix(known, key) {
					continue
				}

				t.Errorf("%s sets %s, which no configuration reads", source.path, key)
			}
		})
	}
}

// TestEnvDistHasNoEmptyDefaultedValues fails when .env.dist sets a variable to
// an empty value although the binary has a non-empty default for it: the empty
// value would override the default.
func TestEnvDistHasNoEmptyDefaultedValues(t *testing.T) {
	configs, err := configInventory()
	if err != nil {
		t.Fatalf("config inventory: %v", err)
	}

	defaults := map[string]string{}
	for key, value := range envDefaults {
		defaults[key] = value
	}

	for _, config := range configs {
		for _, field := range config.Fields {
			if field.DefaultValue != "" {
				defaults[field.Key] = field.DefaultValue
			}
		}
	}

	content, err := os.ReadFile("../.env.dist")
	if err != nil {
		t.Fatalf("read .env.dist: %v", err)
	}

	for _, key := range emptyEnvDistKeysWithDefaults(string(content), defaults) {
		t.Errorf(".env.dist sets %s to an empty value, but its default is %q", key, defaults[key])
	}
}

// TestEnvDistLeavesDockerOwnerIDEmpty fails when .env.dist gives
// DOCKER_EXECUTOR_OWNER_ID a value: every checkout copying it would share that
// owner id on one Docker daemon, and their executors would remove each other's
// containers.
func TestEnvDistLeavesDockerOwnerIDEmpty(t *testing.T) {
	content, err := os.ReadFile("../.env.dist")
	if err != nil {
		t.Fatalf("read .env.dist: %v", err)
	}

	found := false

	for line := range strings.Lines(string(content)) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key != "DOCKER_EXECUTOR_OWNER_ID" {
			continue
		}

		found = true

		if value != "" {
			t.Errorf(".env.dist sets DOCKER_EXECUTOR_OWNER_ID to %q; leave it empty so each checkout sets its own", value)
		}
	}

	if !found {
		t.Error(".env.dist does not list DOCKER_EXECUTOR_OWNER_ID")
	}
}

func TestEmptyEnvDistKeysWithDefaults(t *testing.T) {
	defaults := map[string]string{"A": "x", "B": "y", "C": "z"}
	text := "A=\nB=set\n# C=\nD=\nE= \n"

	got := emptyEnvDistKeysWithDefaults(text, defaults)
	if len(got) != 1 || got[0] != "A" {
		t.Fatalf("got %v, want [A]", got)
	}
}

func emptyEnvDistKeysWithDefaults(text string, defaults map[string]string) []string {
	var keys []string

	for _, match := range regexp.MustCompile(`(?m)^([A-Z][A-Z0-9_]*)=[ \t]*$`).FindAllStringSubmatch(text, -1) {
		if defaults[match[1]] != "" {
			keys = append(keys, match[1])
		}
	}

	return keys
}

func hasKnownPrefix(known map[string]bool, prefix string) bool {
	for key := range known {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// TestConfigInventoryRoles asserts who reads a key: the server alone, the
// executors alone, or both, and that a key is listed once.
func TestConfigInventoryRoles(t *testing.T) {
	groups, err := configInventory()
	if err != nil {
		t.Fatalf("config inventory: %v", err)
	}

	roles := map[string]string{}

	for _, group := range groups {
		for _, field := range group.Fields {
			if _, listed := roles[field.Key]; listed {
				t.Errorf("%s is listed twice", field.Key)
			}

			roles[field.Key] = field.Role
		}
	}

	for key, want := range map[string]string{
		"DB_DSN":                       roleServer,
		"AUTH_JWT_SECRET":              roleServer,
		"WATERMILL_TRANSPORT":          roleServer,
		"PIPELINE_EXECUTOR_API_TOKEN":  roleServer,
		"EXECUTOR_API_URL":             roleExecutor,
		"EXECUTOR_API_TOKEN":           roleExecutor,
		"LOG_LEVEL":                    roleBoth,
		"INTERNAL_HTTP_SERVER_ADDR":    roleBoth,
		"MCP_ENABLED":                  roleServer,
		"MCP_WRITE_TOOLS":              roleServer,
		"MCP_HTTP_SERVER_ADDR":         roleServer,
		"DOCKER_EXECUTOR_OWNER_ID":     roleBoth,
		"K8S_EXECUTOR_NAMESPACE":       roleBoth,
		"K8S_EXECUTOR_OWNER_ID":        roleBoth,
		"K8S_EXECUTOR_OWNER_ID_PREFIX": roleBoth,
		"PIPELINE_RUNTIME_CPU_LIMIT":   roleBoth,
		"PIPELINE_IDLE_TIMEOUT":        roleBoth,
	} {
		if got := roles[key]; got != want {
			t.Errorf("%s: role %q, want %q", key, got, want)
		}
	}
}

// TestControlPlaneOnlyKeysAreServerOnly fails when an executor graph reads a
// variable dropControlPlaneEnv unsets.
func TestControlPlaneOnlyKeysAreServerOnly(t *testing.T) {
	groups, err := configInventory()
	if err != nil {
		t.Fatalf("config inventory: %v", err)
	}

	for _, group := range groups {
		for _, field := range group.Fields {
			if field.Role != roleServer && isControlPlaneOnlyKey(field.Key) {
				t.Errorf("%s is a control-plane variable but an executor reads it (%s)", field.Key, field.Role)
			}
		}
	}
}

func TestDotEnvLine(t *testing.T) {
	for _, tc := range []struct {
		field configField
		value string
		want  string
	}{
		{configField{Key: "LOG_LEVEL", Role: roleBoth}, "info", `LOG_LEVEL="info" # server, executor`},
		{configField{Key: "DB_DSN", Required: true, Role: roleServer}, "", "# DB_DSN= # required, server"},
		{configField{Key: "EXECUTOR_API_URL", Role: roleExecutor}, "", "# EXECUTOR_API_URL= # executor"},
	} {
		if got := dotEnvLine(tc.field, tc.value); got != tc.want {
			t.Errorf("dotEnvLine(%s) = %q, want %q", tc.field.Key, got, tc.want)
		}
	}
}
