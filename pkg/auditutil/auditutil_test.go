package auditutil_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/auditutil"
)

func TestRedact_MasksSecretKeys(t *testing.T) {
	in := map[string]any{
		"host":          "db.example.com",
		"port":          5432,
		"password":      "hunter2",
		"api_token":     "abc123",
		"oauth_token":   "xyz",
		"client_secret": "s3cret",
		"access_key":    "AKIA1234",
		"private_key":   "BEGIN-RSA",
		"credentials":   map[string]any{"username": "u", "password": "p"},
	}

	got := auditutil.Redact(in).(map[string]any)
	assert.Equal(t, "db.example.com", got["host"])
	assert.Equal(t, 5432, got["port"])
	assert.Equal(t, auditutil.RedactMask, got["password"])
	assert.Equal(t, auditutil.RedactMask, got["api_token"])
	assert.Equal(t, auditutil.RedactMask, got["oauth_token"])
	assert.Equal(t, auditutil.RedactMask, got["client_secret"])
	assert.Equal(t, auditutil.RedactMask, got["access_key"])
	assert.Equal(t, auditutil.RedactMask, got["private_key"])
	// Whole nested map under a secret key is masked, not walked.
	assert.Equal(t, auditutil.RedactMask, got["credentials"])
}

func TestRedact_DoesNotMaskNonSecretFields(t *testing.T) {
	in := map[string]any{
		"cursor_key":   "id", // "key" alone is not a secret signal
		"stream_key":   "users",
		"public_field": "ok",
	}

	got := auditutil.Redact(in).(map[string]any)
	assert.Equal(t, "id", got["cursor_key"])
	assert.Equal(t, "users", got["stream_key"])
	assert.Equal(t, "ok", got["public_field"])
}

func TestRedact_WalksNestedStructures(t *testing.T) {
	in := map[string]any{
		"auth": map[string]any{
			"username": "alice",
			"password": "wonderland",
		},
		"items": []any{
			map[string]any{"id": 1, "secret": "leak"},
			map[string]any{"id": 2, "secret": "also-leak"},
		},
	}

	got := auditutil.Redact(in).(map[string]any)
	auth := got["auth"].(map[string]any)
	assert.Equal(t, "alice", auth["username"])
	assert.Equal(t, auditutil.RedactMask, auth["password"])

	items := got["items"].([]any)
	require.Len(t, items, 2)

	first := items[0].(map[string]any)
	assert.Equal(t, 1, first["id"])
	assert.Equal(t, auditutil.RedactMask, first["secret"])
}

func TestRedact_PreservesOriginal(t *testing.T) {
	in := map[string]any{"password": "leak"}
	_ = auditutil.Redact(in)
	assert.Equal(t, "leak", in["password"], "Redact must not mutate input")
}

func TestRedactJSON_RoundtripsAndRedacts(t *testing.T) {
	raw := []byte(`{"host":"db","password":"hunter2","port":5432}`)
	out := auditutil.RedactJSON(raw)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(out, &parsed))
	assert.Equal(t, "db", parsed["host"])
	assert.Equal(t, auditutil.RedactMask, parsed["password"])
	assert.InDelta(t, 5432, parsed["port"], 0.0)
}

func TestRedactJSON_NonJSONInputIsReturnedUnchanged(t *testing.T) {
	raw := []byte("not json")
	out := auditutil.RedactJSON(raw)
	assert.Equal(t, raw, out)
}

func TestDiff_ReportsAddedRemovedAndChangedFields(t *testing.T) {
	before := map[string]any{
		"name": "old",
		"host": "a",
		"port": 1,
	}
	after := map[string]any{
		"name":  "new",
		"host":  "a",
		"extra": "added",
	}

	changes := auditutil.Diff(before, after)
	pathMap := map[string]auditutil.FieldChange{}

	for _, c := range changes {
		pathMap[c.Path] = c
	}

	assert.Equal(t, "old", pathMap["name"].Before)
	assert.Equal(t, "new", pathMap["name"].After)
	assert.Nil(t, pathMap["extra"].Before)
	assert.Equal(t, "added", pathMap["extra"].After)
	assert.InDelta(t, 1, pathMap["port"].Before, 0.0)
	assert.Nil(t, pathMap["port"].After)
	_, hostChanged := pathMap["host"]
	assert.False(t, hostChanged, "host did not change and must not appear")
}

func TestDiff_RedactsSecretFieldsInBothSides(t *testing.T) {
	before := map[string]any{"password": "old-secret"}
	after := map[string]any{"password": "new-secret"}

	changes := auditutil.Diff(before, after)
	require.Len(t, changes, 1)
	assert.Equal(t, "password", changes[0].Path)
	assert.Equal(t, auditutil.RedactMask, changes[0].Before)
	assert.Equal(t, auditutil.RedactMask, changes[0].After)
}

func TestDiff_WalksNestedMapsAndSlices(t *testing.T) {
	before := map[string]any{
		"auth": map[string]any{
			"username": "alice",
		},
		"streams": []any{"a", "b"},
	}
	after := map[string]any{
		"auth": map[string]any{
			"username": "bob",
		},
		"streams": []any{"a", "c", "d"},
	}

	changes := auditutil.Diff(before, after)
	paths := make([]string, 0, len(changes))

	for _, c := range changes {
		paths = append(paths, c.Path)
	}

	assert.Contains(t, paths, "auth.username")
	assert.Contains(t, paths, "streams[1]")
	assert.Contains(t, paths, "streams[2]")
}

func TestTruncateDiff_NoOpUnderTheCap(t *testing.T) {
	small := []auditutil.FieldChange{{Path: "x", Before: 1, After: 2}}
	out, truncated := auditutil.TruncateDiff(small)
	assert.False(t, truncated)
	assert.Equal(t, small, out)
}

func TestTruncateDiff_DropsTailAndAppendsSentinel(t *testing.T) {
	bigValue := strings.Repeat("x", 64)
	changes := make([]auditutil.FieldChange, 0, 200)

	for range 200 {
		changes = append(changes, auditutil.FieldChange{Path: "x", Before: bigValue, After: bigValue})
	}

	out, truncated := auditutil.TruncateDiff(changes)
	assert.True(t, truncated, "200 fat changes should not fit under 8KB")
	require.NotEmpty(t, out)
	last := out[len(out)-1]
	assert.Equal(t, "__truncated__", last.Path)
	assert.Less(t, len(out), len(changes))
}

func TestTruncateDiffWithCap_RespectsCustomCap(t *testing.T) {
	changes := []auditutil.FieldChange{
		{Path: "a", Before: "1", After: "2"},
		{Path: "b", Before: "3", After: "4"},
		{Path: "c", Before: "5", After: "6"},
	}
	out, truncated := auditutil.TruncateDiffWithCap(changes, 60)
	assert.True(t, truncated)
	assert.NotEqual(t, 3, len(out))
}
