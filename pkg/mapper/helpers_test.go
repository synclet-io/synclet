package mapper

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func steps(t *testing.T, raw string) []Step {
	t.Helper()

	var out []Step
	require.NoError(t, json.Unmarshal([]byte(raw), &out))

	return out
}

func mustCompile(t *testing.T, in Input) (*Chain, *Result) {
	t.Helper()

	chain, res := Compile(in)
	require.Empty(t, res.Errors)
	require.NotNil(t, chain)

	return chain, res
}

func record(t *testing.T, raw string) map[string]any {
	t.Helper()

	rec, err := DecodeRecord([]byte(raw))
	require.NoError(t, err)

	return rec
}
