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

func hasKnownPrefix(known map[string]bool, prefix string) bool {
	for key := range known {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}
