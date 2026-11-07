package pipelinesecrets

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// ArrayItemSegment is the path segment that stands for every element of an
// array in a secret path pattern.
const ArrayItemSegment = "*"

// ExtractSecretPaths parses a connector spec JSON and returns the sorted path
// patterns of fields annotated with "airbyte_secret": true. ExpandSecretPaths
// turns them into the concrete paths of a config.
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
// paths to fields with "airbyte_secret": true. Array items appear in a path
// as the ArrayItemSegment segment.
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

			collectFieldSecretPaths(fieldMap, path, paths)
		}
	}

	collectFromBranches(schema, "oneOf", prefix, paths)
	collectFromBranches(schema, "anyOf", prefix, paths)
}

// collectFieldSecretPaths collects the secret paths of one field schema: the
// field itself, its nested properties and branches, and its array items.
func collectFieldSecretPaths(field map[string]any, path string, paths *[]string) {
	if isAirbyteSecret(field) {
		*paths = append(*paths, path)
	}

	collectSecretPaths(field, path, paths)

	if items, ok := field["items"].(map[string]any); ok {
		collectFieldSecretPaths(items, path+"."+ArrayItemSegment, paths)
	}
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

// walkAndReplace recursively walks a JSON object, including every array
// element, and replaces string values.
func walkAndReplace(obj map[string]any, replacer func(string) (string, bool)) {
	for key, value := range obj {
		obj[key] = replaceValue(value, replacer)
	}
}

func replaceValue(value any, replacer func(string) (string, bool)) any {
	switch val := value.(type) {
	case string:
		if newVal, replaced := replacer(val); replaced {
			return newVal
		}
	case map[string]any:
		walkAndReplace(val, replacer)
	case []any:
		for i, item := range val {
			val[i] = replaceValue(item, replacer)
		}
	}

	return value
}

// walkAndCollect recursively walks a JSON object, including every array
// element, and collects string values.
func walkAndCollect(obj map[string]any, collector func(string)) {
	for _, value := range obj {
		collectValue(value, collector)
	}
}

func collectValue(value any, collector func(string)) {
	switch val := value.(type) {
	case string:
		collector(val)
	case map[string]any:
		walkAndCollect(val, collector)
	case []any:
		for _, item := range val {
			collectValue(item, collector)
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

// ExpandSecretPaths resolves secret path patterns against a config: every
// ArrayItemSegment segment becomes the index of each element of the array at
// that position. Only paths present in obj are returned.
func ExpandSecretPaths(obj map[string]any, patterns []string) []string {
	var paths []string

	for _, pattern := range patterns {
		expandPath(obj, strings.Split(pattern, "."), "", &paths)
	}

	return paths
}

func expandPath(current any, parts []string, prefix string, paths *[]string) {
	if len(parts) == 0 {
		*paths = append(*paths, prefix)

		return
	}

	join := func(segment string) string {
		if prefix == "" {
			return segment
		}

		return prefix + "." + segment
	}

	if parts[0] == ArrayItemSegment {
		list, ok := current.([]any)
		if !ok {
			return
		}

		for i, item := range list {
			expandPath(item, parts[1:], join(strconv.Itoa(i)), paths)
		}

		return
	}

	m, ok := current.(map[string]any)
	if !ok {
		return
	}

	child, ok := m[parts[0]]
	if !ok {
		return
	}

	expandPath(child, parts[1:], join(parts[0]), paths)
}

// GetNestedField retrieves a value from a nested map using a dot-notation
// path. A numeric segment indexes into an array.
func GetNestedField(obj map[string]any, path string) (any, bool) {
	current := any(obj)

	for _, part := range strings.Split(path, ".") {
		next, ok := childAt(current, part)
		if !ok {
			return nil, false
		}

		current = next
	}

	return current, true
}

// SetNestedField sets a value in a nested map using a dot-notation path. A
// numeric segment indexes into an array; missing parents leave obj unchanged.
func SetNestedField(obj map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := any(obj)

	for _, part := range parts[:len(parts)-1] {
		next, ok := childAt(current, part)
		if !ok {
			return
		}

		current = next
	}

	last := parts[len(parts)-1]

	switch container := current.(type) {
	case map[string]any:
		container[last] = value
	case []any:
		if i, ok := arrayIndex(container, last); ok {
			container[i] = value
		}
	}
}

func childAt(current any, segment string) (any, bool) {
	switch container := current.(type) {
	case map[string]any:
		val, ok := container[segment]

		return val, ok
	case []any:
		i, ok := arrayIndex(container, segment)
		if !ok {
			return nil, false
		}

		return container[i], true
	default:
		return nil, false
	}
}

func arrayIndex(list []any, segment string) (int, bool) {
	i, err := strconv.Atoi(segment)
	if err != nil || i < 0 || i >= len(list) {
		return 0, false
	}

	return i, true
}
