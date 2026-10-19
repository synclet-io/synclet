package mapper

import (
	"encoding/json"
	"os"
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
		"fallback":{"type":"string"},
		"present":{"type":"string"},
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

const coalesceSchema = `{"type":"object","properties":{
	"nick":{"type":["null","string"]},
	"alias":{"type":["null","string"]},
	"name":{"type":"string"},
	"discount":{"type":["null","number"]},
	"count":{"type":"integer"},
	"any":{},
	"items":{"type":"array"}
}}`

func TestCoalesceTypes(t *testing.T) {
	t.Parallel()

	_, res := mustCompile(t, Input{
		Schema: json.RawMessage(coalesceSchema),
		Steps: steps(t, `[{"set":{
			"str":"coalesce(r.nick, 'anon')",
			"num":"coalesce(r.discount, 0.0)",
			"three":"coalesce(r.nick, r.alias, 'anon')",
			"three_null":"coalesce(r.name, r.nick, r.alias)",
			"pair_null":"coalesce(r.nick, r.alias)",
			"pair_last_set":"coalesce(r.nick, r.name)",
			"dyn":"coalesce(r.any, 'x')"
		}}]`),
	})

	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(res.Schema, &schema))

	for field, want := range map[string]string{
		"str":           `{"type":"string"}`,
		"num":           `{"type":"number"}`,
		"three":         `{"type":"string"}`,
		"three_null":    `{"type":["null","string"]}`,
		"pair_null":     `{"type":["null","string"]}`,
		"pair_last_set": `{"type":"string"}`,
		"dyn":           `{}`,
	} {
		require.JSONEq(t, want, string(schema.Properties[field]), field)
	}
}

func TestCoalesceCompileErrors(t *testing.T) {
	t.Parallel()

	for expr, want := range map[string]string{
		"coalesce(r.discount, 0)":       "coalesce arguments have different types: double and int",
		"coalesce(r.nick, r.alias, 1)":  "coalesce arguments have different types: string and int",
		"coalesce(r.discount, r.count)": "coalesce arguments have different types: double and int",
		"coalesce(r.items, {'a': 1})":   "coalesce arguments have different types: list(dyn) and map(string, int)",
		"coalesce(r.any, r.nick, 1)":    "coalesce arguments have different types: string and int",
		"coalesce(r.nick, null)":        "<input>:1:18: coalesce argument cannot be null",
		"coalesce(r.nick, null, 'x')":   "<input>:1:18: coalesce argument cannot be null",
		"coalesce(r.nick)":              "<input>:1:9: coalesce takes at least two arguments",
		"coalesce()":                    "<input>:1:9: coalesce takes at least two arguments",
	} {
		_, res := Compile(Input{
			Schema: json.RawMessage(coalesceSchema),
			Steps:  steps(t, `[{"set":{"x":"`+expr+`"}}]`),
		})
		require.Len(t, res.Errors, 1, expr)
		require.Equal(t, 0, res.Errors[0].Step, expr)
		require.Contains(t, res.Errors[0].Message, want, expr)
		require.NotContains(t, res.Errors[0].Message, coalesceFunction, expr)
	}
}

func TestCoalesceRuntime(t *testing.T) {
	t.Parallel()

	chain, _ := mustCompile(t, Input{
		Schema: json.RawMessage(coalesceSchema),
		Steps: steps(t, `[{"set":{
			"pair":"coalesce(r.nick, 'anon')",
			"three":"coalesce(r.nick, r.alias, 'anon')"
		}}]`),
	})

	for rec, want := range map[string][2]string{
		`{}`:                         {"anon", "anon"},
		`{"nick":null,"alias":null}`: {"anon", "anon"},
		`{"alias":"al"}`:             {"anon", "al"},
		`{"nick":null,"alias":"al"}`: {"anon", "al"},
		`{"nick":"n","alias":"al"}`:  {"n", "n"},
	} {
		out, _, err := chain.Run(record(t, rec), testSync)
		require.NoError(t, err, rec)
		require.Equal(t, want[0], out["pair"], rec)
		require.Equal(t, want[1], out["three"], rec)
	}
}

// TestEditorFunctions compiles the example of every function the front-end
// expression editor completes, so the editor offers only what the engine defines.
func TestEditorFunctions(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../front/src/features/stream-config/transform/cel/functions.json")
	require.NoError(t, err)

	var functions []struct {
		Name    string `json:"name"`
		Example string `json:"example"`
	}
	require.NoError(t, json.Unmarshal(raw, &functions))
	require.NotEmpty(t, functions)

	schema := json.RawMessage(`{"type":"object","properties":{"s":{"type":"string"},"opt":{"type":["null","string"]},"n":{"type":"number"}}}`)

	for _, fn := range functions {
		require.Contains(t, fn.Example, fn.Name+"(", "example of %s does not call it", fn.Name)

		stepJSON, err := json.Marshal([]map[string]map[string]string{{"set": {"out": fn.Example}}})
		require.NoError(t, err)

		_, res := Compile(Input{Schema: schema, Steps: steps(t, string(stepJSON))})
		require.Empty(t, res.Errors, "%s: %s", fn.Name, fn.Example)
	}
}
