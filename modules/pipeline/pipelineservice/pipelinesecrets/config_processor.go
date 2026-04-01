package pipelinesecrets

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// ExtractSecretPaths parses a connector spec JSON and returns sorted paths
// of fields annotated with "airbyte_secret": true.
func ExtractSecretPaths(specJSON string) ([]string, error) {
	var spec map[string]any
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return nil, fmt.Errorf("parsing spec JSON: %w", err)
	}

	connSpec, ok := spec["connectionSpecification"].(map[string]any)
	if !ok {
		return nil, nil
	}

	var paths []string
	collectSecretPaths(connSpec, "", &paths)

	sort.Strings(paths)

	return uniqueStrings(paths), nil
}

// MaskConfigSecrets replaces all $secret:uuid values in config JSON with the mask placeholder.
// Does not need the spec -- pattern-matches on $secret: prefix.
func MaskConfigSecrets(configJSON string) (string, error) {
	var configMap map[string]any
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return "", fmt.Errorf("parsing config JSON: %w", err)
	}

	walkAndReplace(configMap, func(value string) (string, bool) {
		if secretutil.IsSecretRef(value) {
			return secretutil.SecretMask, true
		}

		return value, false
	})

	result, err := json.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	return string(result), nil
}

// ExtractSecretRefs extracts all secret reference UUIDs from a config JSON.
func ExtractSecretRefs(configJSON string) ([]uuid.UUID, error) {
	var configMap map[string]any
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	var refs []uuid.UUID

	walkAndCollect(configMap, func(value string) {
		if secretutil.IsSecretRef(value) {
			if id, err := secretutil.ExtractSecretID(value); err == nil {
				refs = append(refs, id)
			}
		}
	})

	return refs, nil
}

// collectSecretPaths recursively walks a JSON schema object and collects
// paths to fields with "airbyte_secret": true.
func collectSecretPaths(schema map[string]any, prefix string, paths *[]string) {
	props, ok := schema["properties"].(map[string]any)
	if ok {
		for fieldName, fieldSchema := range props {
			fieldMap, ok := fieldSchema.(map[string]any)
			if !ok {
				continue
			}

			path := fieldName
			if prefix != "" {
				path = prefix + "." + fieldName
			}

			if isAirbyteSecret(fieldMap) {
				*paths = append(*paths, path)
			}

			// Recurse into nested objects
			if _, hasProps := fieldMap["properties"]; hasProps {
				collectSecretPaths(fieldMap, path, paths)
			}

			// Handle oneOf/anyOf
			collectFromBranches(fieldMap, "oneOf", path, paths)
			collectFromBranches(fieldMap, "anyOf", path, paths)
		}
	}

	// Handle oneOf/anyOf at the current schema level
	collectFromBranches(schema, "oneOf", prefix, paths)
	collectFromBranches(schema, "anyOf", prefix, paths)
}

// collectFromBranches collects secret paths from oneOf/anyOf branches.
func collectFromBranches(schema map[string]any, key, prefix string, paths *[]string) {
	branches, ok := schema[key].([]any)
	if !ok {
		return
	}

	for _, branch := range branches {
		branchMap, ok := branch.(map[string]any)
		if !ok {
			continue
		}

		collectSecretPaths(branchMap, prefix, paths)
	}
}

func isAirbyteSecret(field map[string]any) bool {
	secret, ok := field["airbyte_secret"]
	if !ok {
		return false
	}

	b, ok := secret.(bool)

	return ok && b
}

// walkAndReplace recursively walks a JSON object and replaces string values.
func walkAndReplace(obj map[string]any, replacer func(string) (string, bool)) {
	for key, value := range obj {
		switch val := value.(type) {
		case string:
			if newVal, replaced := replacer(val); replaced {
				obj[key] = newVal
			}
		case map[string]any:
			walkAndReplace(val, replacer)
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					walkAndReplace(m, replacer)
				}
			}
		}
	}
}

// walkAndCollect recursively walks a JSON object and collects string values.
func walkAndCollect(obj map[string]any, collector func(string)) {
	for _, value := range obj {
		switch val := value.(type) {
		case string:
			collector(val)
		case map[string]any:
			walkAndCollect(val, collector)
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					walkAndCollect(m, collector)
				}
			}
		}
	}
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return items
	}

	seen := make(map[string]bool, len(items))

	result := make([]string, 0, len(items))
	for _, val := range items {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}

	return result
}

// GetNestedField retrieves a value from a nested map using a dot-notation path.
func GetNestedField(obj map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := any(obj)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}

// SetNestedField sets a value in a nested map using a dot-notation path.
func SetNestedField(obj map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			return
		}

		current = next
	}

	current[parts[len(parts)-1]] = value
}
