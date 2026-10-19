package mapper

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCostLimit(t *testing.T) {
	t.Parallel()

	items := make([]string, 200)
	for i := range items {
		items[i] = "1"
	}

	rec := `{"items":[` + strings.Join(items, ",") + `]}`
	stepsJSON := `[{"set":{"n":"r.items.map(x, x * 2).filter(x, x > 1).size()"}}]`

	chain, _ := mustCompile(t, Input{Schema: json.RawMessage(ordersSchema), Steps: steps(t, stepsJSON), CostLimit: 10})
	_, _, err := chain.Run(record(t, rec), testSync)
	require.ErrorContains(t, err, "cost limit")

	chain, _ = mustCompile(t, Input{Schema: json.RawMessage(ordersSchema), Steps: steps(t, stepsJSON)})
	out, _, err := chain.Run(record(t, rec), testSync)
	require.NoError(t, err)
	require.Equal(t, int64(200), out["n"])
}

func TestFunctions(t *testing.T) {
	t.Parallel()

	chain, res := mustCompile(t, Input{
		Schema: json.RawMessage(`{"type":"object","properties":{"s":{"type":"string"},"opt":{"type":["null","string"]}}}`),
		Steps: steps(t, `[{"set":{
			"ts":"parseTimestamp('02.01.2026 10:00', 'dd.MM.yyyy HH:mm')",
			"ts_tz":"parseTimestamp('02.01.2026 10:00', 'dd.MM.yyyy HH:mm', 'Europe/Kyiv')",
			"ts_unix":"parseTimestamp('1767348000', 'unix')",
			"num":"parseNumber('1.234,5', ',')",
			"num_dot":"parseNumber('1,234.5', '.')",
			"hash":"sha256('abc')",
			"fallback":"coalesce(r.opt, 'none')",
			"present":"coalesce(r.s, 'none')",
			"recent":"now() > timestamp('2026-01-01T00:00:00Z')"
		}}]`),
	})

	require.JSONEq(t, `{"type":"object","properties":{
		"s":{"type":"string"},
		"opt":{"type":["null","string"]},
		"ts":{"type":"string","format":"date-time","airbyte_type":"timestamp_with_timezone"},
		"ts_tz":{"type":"string","format":"date-time","airbyte_type":"timestamp_with_timezone"},
		"ts_unix":{"type":"string","format":"date-time","airbyte_type":"timestamp_with_timezone"},
		"num":{"type":"number"},
		"num_dot":{"type":"number"},
		"hash":{"type":"string"},
		"fallback":{},
		"present":{},
		"recent":{"type":"boolean"}
	}}`, string(res.Schema))

	out, keep, err := chain.Run(record(t, `{"s":"x","opt":null}`), testSync)
	require.NoError(t, err)
	require.True(t, keep)
	require.Equal(t, map[string]any{
		"s":        "x",
		"opt":      nil,
		"ts":       "2026-01-02T10:00:00Z",
		"ts_tz":    "2026-01-02T10:00:00+02:00",
		"ts_unix":  "2026-01-02T10:00:00Z",
		"num":      1234.5,
		"num_dot":  1234.5,
		"hash":     "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		"fallback": "none",
		"present":  "x",
		"recent":   true,
	}, out)

	out, _, err = chain.Run(record(t, `{"s":"x"}`), testSync)
	require.NoError(t, err)
	require.Equal(t, "none", out["fallback"])

	_, res = Compile(Input{
		Schema: json.RawMessage(`{"type":"object","properties":{"s":{"type":"string"}}}`),
		Steps:  steps(t, `[{"set":{"x":"parseTimestamp(r.s, 'QQ')"}}]`),
	})
	require.Len(t, res.Errors, 1)
	require.Contains(t, res.Errors[0].Message, `unknown cast format "QQ"`)
}
