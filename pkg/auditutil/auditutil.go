// Package auditutil provides pure helpers for the upcoming audit module:
// secret redaction over arbitrary JSON-like payloads, structural field diffs
// between "before" and "after" snapshots, and size-bounded truncation so a
// rogue payload cannot blow up the audit table.
//
// The audit module itself (modules/audit) is not yet implemented — see
// docs/roadmap-6m/EPIC-3-audit-log/EPIC.md. This package is the foundation
// it will consume.
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

// Redact returns a deep copy of v with values under secret-looking keys
// replaced by RedactMask. Works recursively on map[string]any and
// []any structures, leaving primitives untouched.
func Redact(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))

		for key, val := range typed {
			if isSecretKey(key) {
				if val == nil {
					out[key] = nil
				} else {
					out[key] = RedactMask
				}

				continue
			}

			out[key] = Redact(val)
		}

		return out
	case []any:
		out := make([]any, len(typed))
		for i, val := range typed {
			out[i] = Redact(val)
		}

		return out
	default:
		return value
	}
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
// changed") but the emitted Before / After values for secret-keyed paths are
// replaced with RedactMask so the actual secret never lands in the audit
// table. The path uses dot notation, with [i] for array indices.
func Diff(before, after any) []FieldChange {
	out := []FieldChange{}
	diffWalk("", "", before, after, &out)

	return out
}

func diffWalk(path, lastKey string, before, after any, out *[]FieldChange) {
	if reflectiveEqual(before, after) {
		return
	}

	beforeMap, beforeOk := before.(map[string]any)
	afterMap, afterOk := after.(map[string]any)

	if beforeOk && afterOk {
		keys := make(map[string]struct{}, len(beforeMap)+len(afterMap))
		for k := range beforeMap {
			keys[k] = struct{}{}
		}

		for k := range afterMap {
			keys[k] = struct{}{}
		}

		for k := range keys {
			child := joinPath(path, k)
			diffWalk(child, k, beforeMap[k], afterMap[k], out)
		}

		return
	}

	beforeSlice, beforeSliceOk := before.([]any)
	afterSlice, afterSliceOk := after.([]any)

	if beforeSliceOk && afterSliceOk {
		maxLen := len(beforeSlice)
		if len(afterSlice) > maxLen {
			maxLen = len(afterSlice)
		}

		for i := range maxLen {
			var beforeItem, afterItem any
			if i < len(beforeSlice) {
				beforeItem = beforeSlice[i]
			}

			if i < len(afterSlice) {
				afterItem = afterSlice[i]
			}

			diffWalk(fmt.Sprintf("%s[%d]", path, i), lastKey, beforeItem, afterItem, out)
		}

		return
	}

	emitBefore := before
	emitAfter := after

	if isSecretKey(lastKey) {
		if emitBefore != nil {
			emitBefore = RedactMask
		}

		if emitAfter != nil {
			emitAfter = RedactMask
		}
	}

	*out = append(*out, FieldChange{Path: path, Before: emitBefore, After: emitAfter})
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
