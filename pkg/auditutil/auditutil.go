// Package auditutil provides pure helpers for the audit module: secret
// redaction over arbitrary JSON-like payloads, structural field diffs between
// "before" and "after" snapshots, and size-bounded truncation so a rogue
// payload cannot blow up the audit table.
package auditutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RedactMask is the placeholder written in place of a redacted value.
const RedactMask = "***REDACTED***"

// secretKeySubstrings lists substrings that, when found (case-insensitively)
// in a field name, force redaction of that field's value. Order does not
// matter; matching is OR.
var secretKeySubstrings = []string{
	"password",
	"secret",
	"token",
	"credential",
	"private_key",
	"privatekey",
	"api_key",
	"apikey",
	"access_key",
	"accesskey",
}

// isSecretKey reports whether a field name should be redacted.
// "key" alone is intentionally NOT in the list — too many false positives
// (cursor_key, stream_key, etc.). Composed names like access_key match.
func isSecretKey(name string) bool {
	lower := strings.ToLower(name)
	for _, s := range secretKeySubstrings {
		if strings.Contains(lower, s) {
			return true
		}
	}

	return false
}

// Redact returns a deep copy of value with the values under secret-looking
// keys, and under every dot-separated secretPaths entry, replaced by
// RedactMask. Works recursively on map[string]any and []any structures,
// leaving primitives untouched.
func Redact(value any, secretPaths ...string) any {
	out := redactKeys(value)

	if root, ok := out.(map[string]any); ok {
		for _, path := range secretPaths {
			maskPath(root, strings.Split(path, "."))
		}
	}

	return out
}

func redactKeys(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))

		for key, val := range typed {
			if isSecretKey(key) {
				out[key] = mask(val)

				continue
			}

			out[key] = redactKeys(val)
		}

		return out
	case []any:
		out := make([]any, len(typed))
		for i, val := range typed {
			out[i] = redactKeys(val)
		}

		return out
	default:
		return value
	}
}

func maskPath(obj map[string]any, parts []string) {
	val, ok := obj[parts[0]]
	if !ok {
		return
	}

	if len(parts) == 1 {
		obj[parts[0]] = mask(val)

		return
	}

	if child, ok := val.(map[string]any); ok {
		maskPath(child, parts[1:])
	}
}

func mask(val any) any {
	if val == nil {
		return nil
	}

	return RedactMask
}

// RedactJSON parses raw JSON, redacts secret fields, and re-marshals.
// Non-JSON input is returned unchanged.
func RedactJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}

	redacted := Redact(v)

	out, err := json.Marshal(redacted)
	if err != nil {
		return raw
	}

	return out
}

// FieldChange records a single field-level change in a diff.
type FieldChange struct {
	Path   string `json:"path"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

// Diff produces a flat list of FieldChange entries between two structurally
// similar payloads. The walk operates on raw values so that genuine changes
// to secret fields still surface (the consumer can tell "the password
// changed") but the emitted Before / After values at or below a secret-keyed
// path are replaced with RedactMask so the actual secret never lands in the
// audit table. A map or slice present on one side only is walked against an
// empty counterpart, so each of its leaves becomes its own change. The path
// uses dot notation, with [i] for array indices.
func Diff(before, after any) []FieldChange {
	out := []FieldChange{}
	diffWalk("", false, before, after, &out)

	return out
}

func diffWalk(path string, secret bool, before, after any, out *[]FieldChange) {
	if reflectiveEqual(before, after) {
		return
	}

	beforeMap, beforeOk := before.(map[string]any)
	afterMap, afterOk := after.(map[string]any)

	if (beforeOk || before == nil) && (afterOk || after == nil) && (beforeOk || afterOk) {
		keys := make(map[string]struct{}, len(beforeMap)+len(afterMap))
		for k := range beforeMap {
			keys[k] = struct{}{}
		}

		for k := range afterMap {
			keys[k] = struct{}{}
		}

		emitted := len(*out)

		for k := range keys {
			diffWalk(joinPath(path, k), secret || isSecretKey(k), beforeMap[k], afterMap[k], out)
		}

		if len(*out) == emitted {
			*out = append(*out, FieldChange{Path: path, Before: before, After: after})
		}

		return
	}

	beforeSlice, beforeSliceOk := before.([]any)
	afterSlice, afterSliceOk := after.([]any)

	if (beforeSliceOk || before == nil) && (afterSliceOk || after == nil) && (beforeSliceOk || afterSliceOk) {
		maxLen := max(len(beforeSlice), len(afterSlice))
		emitted := len(*out)

		for i := range maxLen {
			var beforeItem, afterItem any
			if i < len(beforeSlice) {
				beforeItem = beforeSlice[i]
			}

			if i < len(afterSlice) {
				afterItem = afterSlice[i]
			}

			diffWalk(fmt.Sprintf("%s[%d]", path, i), secret, beforeItem, afterItem, out)
		}

		if len(*out) == emitted {
			*out = append(*out, FieldChange{Path: path, Before: before, After: after})
		}

		return
	}

	if secret {
		before = mask(before)
		after = mask(after)
	}

	*out = append(*out, FieldChange{Path: path, Before: redactKeys(before), After: redactKeys(after)})
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}

	return parent + "." + child
}

// reflectiveEqual is a JSON-friendly equality check that handles maps and slices.
func reflectiveEqual(a, b any) bool {
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)

	if errA != nil || errB != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}

// MaxDiffBytes is the default cap applied by TruncateDiff.
const MaxDiffBytes = 8 * 1024

// TruncateDiff caps the JSON-encoded size of a diff at MaxDiffBytes. If the
// encoded diff exceeds the cap, entries are dropped from the tail and a
// sentinel marker is appended so the consumer knows the truncation happened.
// Returns the (possibly truncated) slice and a flag indicating truncation.
func TruncateDiff(changes []FieldChange) ([]FieldChange, bool) {
	return TruncateDiffWithCap(changes, MaxDiffBytes)
}

// TruncateDiffWithCap is TruncateDiff with an explicit byte cap, useful for
// tests.
func TruncateDiffWithCap(changes []FieldChange, capBytes int) ([]FieldChange, bool) {
	if capBytes <= 0 {
		return changes, false
	}

	if encoded, err := json.Marshal(changes); err == nil && len(encoded) <= capBytes {
		return changes, false
	}

	// Drop entries from the tail until the encoded size fits.
	truncated := make([]FieldChange, len(changes))
	copy(truncated, changes)

	for len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
		withMarker := make([]FieldChange, 0, len(truncated)+1)
		withMarker = append(withMarker, truncated...)
		withMarker = append(withMarker, FieldChange{Path: "__truncated__", After: fmt.Sprintf("%d changes omitted", len(changes)-len(truncated))})

		if encoded, err := json.Marshal(withMarker); err == nil && len(encoded) <= capBytes {
			return withMarker, true
		}
	}

	return []FieldChange{{Path: "__truncated__", After: fmt.Sprintf("%d changes omitted", len(changes))}}, true
}
