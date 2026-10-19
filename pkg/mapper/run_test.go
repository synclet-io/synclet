package mapper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

var testSync = SyncInfo{
	StartedAt:  time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
	JobID:      "job-1",
	Connection: "conn-1",
}

func runOne(t *testing.T, schema, stepsJSON, rec string) (out map[string]any, keep bool, err error) {
	t.Helper()

	chain, _ := mustCompile(t, Input{Schema: json.RawMessage(schema), Steps: steps(t, stepsJSON), Salt: []byte("s")})

	return chain.Run(record(t, rec), testSync)
}

func hexHash(s string) string {
	sum := sha256.Sum256([]byte(s))

	return hex.EncodeToString(sum[:])
}

func TestRunMappers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps string
		rec   string
		want  map[string]any
	}{
		{
			"rename",
			`[{"rename":{"cust_id":"customer_id"}}]`,
			`{"id":1,"cust_id":7}`,
			map[string]any{"id": int64(1), "customer_id": int64(7)},
		},
		{
			"each step sees the previous one",
			`[{"rename":{"cust_id":"customer_id"}},{"set":{"double":"r.customer_id * 2"}}]`,
			`{"id":1,"cust_id":7}`,
			map[string]any{"id": int64(1), "customer_id": int64(7), "double": int64(14)},
		},
		{
			"set constant, expression and sync metadata",
			`[{"set":{"currency":"'USD'","total_cents":"int(r.amount * 100.0)","email_domain":"r.email.split('@')[1].lowerAscii()","synced_at":"sync.started_at","job":"sync.job_id + '/' + sync.connection"}}]`,
			`{"id":1,"amount":12.5,"email":"a@Example.COM"}`,
			map[string]any{
				"id": int64(1), "amount": 12.5, "email": "a@Example.COM",
				"currency": "USD", "total_cents": int64(1250), "email_domain": "example.com",
				"synced_at": "2026-10-19T12:00:00Z", "job": "job-1/conn-1",
			},
		},
		{
			"set fields read the step input",
			`[{"set":{"amount":"r.amount * 2.0","copy":"r.amount"}}]`,
			`{"id":1,"amount":1.5}`,
			map[string]any{"id": int64(1), "amount": 3.0, "copy": 1.5},
		},
		{
			"set builds lists and maps",
			`[{"set":{"pair":"[r.id, 2]","obj":"{'a': r.id}"}}]`,
			`{"id":1}`,
			map[string]any{"id": int64(1), "pair": []any{int64(1), int64(2)}, "obj": map[string]any{"a": int64(1)}},
		},
		{
			"hash with salt, numbers as JSON text, null kept",
			`[{"hash":["email","id","amount","code"]}]`,
			`{"id":42,"amount":1.5,"email":"a@b.c","code":null}`,
			map[string]any{"id": hexHash("s42"), "amount": hexHash("s1.5"), "email": hexHash("sa@b.c"), "code": nil},
		},
		{
			"mask keep last and keep first",
			`[{"mask":{"email":{"keep_last":4},"code":{"keep_first":1,"keep_last":1}}}]`,
			`{"email":"4111111111111111","code":"ключ"}`,
			map[string]any{"email": "************1111", "code": "к**ч"},
		},
		{
			"mask shorter than kept part",
			`[{"mask":{"code":{"keep_first":3,"keep_last":3}}}]`,
			`{"code":"abc"}`,
			map[string]any{"code": "abc"},
		},
		{
			"filter keeps",
			`[{"filter":"r.status != 'test'"}]`,
			`{"status":"live"}`,
			map[string]any{"status": "live"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, keep, err := runOne(t, ordersSchema, tt.steps, tt.rec)
			require.NoError(t, err)
			require.True(t, keep)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestRunFilterDrops(t *testing.T) {
	t.Parallel()

	_, keep, err := runOne(t, ordersSchema, `[{"filter":"r.status != 'test'"}]`, `{"status":"test"}`)
	require.NoError(t, err)
	require.False(t, keep)
}

func TestRunCast(t *testing.T) {
	t.Parallel()

	const schema = `{"type":"object","properties":{
		"v":{"type":"string"},
		"n":{"type":"number"},
		"i":{"type":"integer"},
		"ts":{"type":"string","format":"date-time"},
		"code":{"type":"string"}
	}}`

	tests := []struct {
		name string
		cast string
		rec  string
		want any
	}{
		{"number with comma decimal and dot thousands", `{"to":"number","decimal_separator":",","thousands_separator":"."}`, `{"v":"1.234,5"}`, 1234.5},
		{"bare target string", `"number"`, `{"v":"3.5"}`, 3.5},
		{"integer with trim and prefix", `{"to":"integer","trim":true,"strip_prefix":"$"}`, `{"v":" $12 "}`, int64(12)},
		{"number with suffix", `{"to":"number","strip_suffix":"%"}`, `{"v":"12.5%"}`, 12.5},
		{"integer from integral number", `"integer"`, `{"v":"7"}`, int64(7)},
		{"boolean with custom values", `{"to":"boolean","true_values":["Y"],"false_values":["N"]}`, `{"v":"Y"}`, true},
		{"boolean false with custom values", `{"to":"boolean","true_values":["Y"],"false_values":["N"]}`, `{"v":"N"}`, false},
		{"boolean default values", `"boolean"`, `{"v":"true"}`, true},
		{"timestamp pattern with timezone", `{"to":"timestamp","format":"dd.MM.yyyy HH:mm","timezone":"Europe/Kyiv"}`, `{"v":"02.01.2026 10:00"}`, "2026-01-02T10:00:00+02:00"},
		{"timestamp pattern with seconds and millis", `{"to":"timestamp","format":"yyyy-MM-dd'T'HH:mm:ss.SSS"}`, `{"v":"2026-01-02T10:00:05.123"}`, "2026-01-02T10:00:05.123Z"},
		{"timestamp iso8601 default", `"timestamp"`, `{"v":"2026-01-02T10:00:00+03:00"}`, "2026-01-02T10:00:00+03:00"},
		{"timestamp iso8601 without zone uses timezone", `{"to":"timestamp","timezone":"Europe/Kyiv"}`, `{"v":"2026-01-02T10:00:00"}`, "2026-01-02T10:00:00+02:00"},
		{"timestamp unix seconds", `{"to":"timestamp","format":"unix"}`, `{"v":"1767348000"}`, "2026-01-02T10:00:00Z"},
		{"timestamp unix_ms from number", `{"to":"timestamp","format":"unix_ms"}`, `{"n":1767348000000}`, "2026-01-02T10:00:00Z"},
		{"date pattern", `{"to":"date","format":"dd/MM/yyyy"}`, `{"v":"02/01/2026"}`, "2026-01-02"},
		{"object from JSON string", `"object"`, `{"v":"{\"a\":1,\"b\":[true]}"}`, map[string]any{"a": int64(1), "b": []any{true}}},
		{"array from JSON string", `"array"`, `{"v":"[1,\"x\"]"}`, []any{int64(1), "x"}},
		{"number to string", `"string"`, `{"n":12.5}`, "12.5"},
		{"number to string with decimal separator", `{"to":"string","decimal_separator":","}`, `{"n":12.5}`, "12,5"},
		{"integer to string", `"string"`, `{"i":42}`, "42"},
		{"timestamp to string with pattern", `{"to":"string","format":"dd.MM.yyyy"}`, `{"ts":"2026-01-02T10:00:00Z"}`, "02.01.2026"},
		{"integer to number", `"number"`, `{"i":3}`, 3.0},
		{"null stays null", `"integer"`, `{"v":null}`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var rec map[string]any
			require.NoError(t, json.Unmarshal([]byte(tt.rec), &rec))

			field := ""
			for k := range rec {
				field = k
			}

			out, keep, err := runOne(t, schema, `[{"cast":{"`+field+`":`+tt.cast+`}}]`, tt.rec)
			require.NoError(t, err)
			require.True(t, keep)
			require.Equal(t, tt.want, out[field])
		})
	}
}

func TestRunCastSource(t *testing.T) {
	t.Parallel()

	out, keep, err := runOne(t, ordersSchema,
		`[{"cast":{"code_num":{"to":"integer","source":"r.code.split('-')[0]"}}},{"set":{"next":"r.code_num + 1"}}]`,
		`{"code":"42-a"}`)
	require.NoError(t, err)
	require.True(t, keep)
	require.Equal(t, map[string]any{"code": "42-a", "code_num": int64(42), "next": int64(43)}, out)
}

func TestRunOnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps string
		rec   string
		field string
	}{
		{"set", `[{"set":{"x":"int(r.code)"},"on_error":"%s"}]`, `{"code":"abc"}`, "x"},
		{"cast", `[{"cast":{"code":{"to":"integer","on_error":"%s"}}}]`, `{"code":"abc"}`, "code"},
		{"cast step default", `[{"cast":{"code":"integer"},"on_error":"%s"}]`, `{"code":"abc"}`, "code"},
		{"cast source", `[{"cast":{"y":{"to":"integer","source":"r.code.split('-')[1]","on_error":"%s"}}}]`, `{"code":"abc"}`, "y"},
		{"hash of an object", `[{"hash":["address"],"on_error":"%s"}]`, `{"address":{"city":"Kyiv"}}`, "address"},
		{"mask of a list", `[{"mask":{"items":{"keep_last":1}},"on_error":"%s"}]`, `{"items":[1,2]}`, "items"},
		{"set over a missing field", `[{"set":{"x":"r.amount * 2.0"},"on_error":"%s"}]`, `{"id":1}`, "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := runOne(t, ordersSchema, fmt.Sprintf(tt.steps, "fail"), tt.rec)

			var stepErr *StepError
			require.ErrorAs(t, err, &stepErr)
			require.Equal(t, 0, stepErr.Step)
			require.Equal(t, tt.field, stepErr.Field)

			out, keep, err := runOne(t, ordersSchema, fmt.Sprintf(tt.steps, "null"), tt.rec)
			require.NoError(t, err)
			require.True(t, keep)
			require.Contains(t, out, tt.field)
			require.Nil(t, out[tt.field])

			_, keep, err = runOne(t, ordersSchema, fmt.Sprintf(tt.steps, "skip"), tt.rec)
			require.NoError(t, err)
			require.False(t, keep)
		})
	}

	t.Run("filter", func(t *testing.T) {
		t.Parallel()

		_, _, err := runOne(t, ordersSchema, `[{"filter":"r.id / r.cust_id > 1"}]`, `{"id":1,"cust_id":0}`)

		var stepErr *StepError
		require.ErrorAs(t, err, &stepErr)
		require.Equal(t, 0, stepErr.Step)

		_, keep, err := runOne(t, ordersSchema, `[{"filter":"r.id / r.cust_id > 1","on_error":"skip"}]`, `{"id":1,"cust_id":0}`)
		require.NoError(t, err)
		require.False(t, keep)
	})

	t.Run("step index of a later step", func(t *testing.T) {
		t.Parallel()

		_, _, err := runOne(t, ordersSchema, `[{"set":{"a":"1"}},{"cast":{"code":"integer"}}]`, `{"code":"abc"}`)

		var stepErr *StepError
		require.ErrorAs(t, err, &stepErr)
		require.Equal(t, 1, stepErr.Step)
		require.ErrorIs(t, err, stepErr.Err)
	})
}

func TestFieldSelection(t *testing.T) {
	t.Parallel()

	in := Input{
		Schema: json.RawMessage(ordersSchema),
		Steps:  steps(t, `[{"set":{"total_cents":"int(r.amount * 100.0)"}},{"hash":["email"]}]`),
		Salt:   []byte("s"),
		SelectedFields: []protocol.SelectedField{
			{FieldPath: []string{"id"}},
			{FieldPath: []string{"address", "city"}},
		},
	}

	chain, res := mustCompile(t, in)
	require.JSONEq(t, `{"type":"object","properties":{
		"id":{"type":"integer"},
		"address":{"type":"object","properties":{"city":{"type":"string"}}},
		"total_cents":{"type":["null","integer"]},
		"email":{"type":"string"}
	}}`, string(res.Schema))

	out, keep, err := chain.Run(record(t, `{"id":1,"amount":2.5,"email":"a@b.c","status":"x","address":{"city":"Kyiv","zip":"01001"}}`), testSync)
	require.NoError(t, err)
	require.True(t, keep)
	require.Equal(t, map[string]any{
		"id":          int64(1),
		"address":     map[string]any{"city": "Kyiv"},
		"total_cents": int64(250),
		"email":       hexHash("sa@b.c"),
	}, out)

	t.Run("empty selection keeps everything", func(t *testing.T) {
		t.Parallel()

		chain, _ := mustCompile(t, Input{Schema: json.RawMessage(ordersSchema), Steps: steps(t, `[{"set":{"x":"1"}}]`)})
		out, _, err := chain.Run(record(t, `{"id":1,"status":"x"}`), testSync)
		require.NoError(t, err)
		require.Equal(t, map[string]any{"id": int64(1), "status": "x", "x": int64(1)}, out)
	})
}

func TestRunIntegerFieldWithFractionZero(t *testing.T) {
	t.Parallel()

	out, _, err := runOne(t, ordersSchema, `[{"set":{"next":"r.id + 1"}}]`, `{"id":1.0}`)
	require.NoError(t, err)
	require.Equal(t, int64(2), out["next"])
	require.Equal(t, int64(1), out["id"])
}

func TestRunRenameSwap(t *testing.T) {
	t.Parallel()

	const rec = `{"id":1,"email":"e","address":{"city":"K","zip":"Z"}}`

	sel := func(paths ...[]string) []protocol.SelectedField {
		out := make([]protocol.SelectedField, 0, len(paths))
		for _, path := range paths {
			out = append(out, protocol.SelectedField{FieldPath: path})
		}

		return out
	}

	addressSchema := `{"type":"object","properties":{"city":{"type":"string"},"zip":{"type":"string"}}}`
	citySchema := `{"type":"object","properties":{"city":{"type":"string"}}}`

	tests := []struct {
		name      string
		steps     string
		selection []protocol.SelectedField
		want      map[string]any
		props     string
	}{
		{
			"swap, no selection",
			`[{"rename":{"address":"email","email":"address"}}]`,
			nil,
			map[string]any{"id": int64(1), "email": map[string]any{"city": "K", "zip": "Z"}, "address": "e"},
			`"email":` + addressSchema + `,"address":{"type":"string"}`,
		},
		{
			"swap, nested selection",
			`[{"rename":{"address":"email","email":"address"}}]`,
			sel([]string{"id"}, []string{"address", "city"}, []string{"email"}),
			map[string]any{"id": int64(1), "email": map[string]any{"city": "K"}, "address": "e"},
			`"email":` + citySchema + `,"address":{"type":"string"}`,
		},
		{
			"swap, whole selection",
			`[{"rename":{"address":"email","email":"address"}}]`,
			sel([]string{"id"}, []string{"address"}, []string{"email"}),
			map[string]any{"id": int64(1), "email": map[string]any{"city": "K", "zip": "Z"}, "address": "e"},
			`"email":` + addressSchema + `,"address":{"type":"string"}`,
		},
		{
			"one way, no selection",
			`[{"rename":{"address":"email","email":"x"}}]`,
			nil,
			map[string]any{"id": int64(1), "email": map[string]any{"city": "K", "zip": "Z"}, "x": "e"},
			`"email":` + addressSchema + `,"x":{"type":"string"}`,
		},
		{
			"one way, nested selection",
			`[{"rename":{"address":"email","email":"x"}}]`,
			sel([]string{"id"}, []string{"address", "city"}, []string{"email"}),
			map[string]any{"id": int64(1), "email": map[string]any{"city": "K"}, "x": "e"},
			`"email":` + citySchema + `,"x":{"type":"string"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema := `{"type":"object","properties":{"id":{"type":"integer"},"email":{"type":"string"},"address":` + addressSchema + `}}`
			chain, res := mustCompile(t, Input{Schema: json.RawMessage(schema), Steps: steps(t, tt.steps), SelectedFields: tt.selection})
			require.JSONEq(t, `{"type":"object","properties":{"id":{"type":"integer"},`+tt.props+`}}`, string(res.Schema))

			out, keep, err := chain.Run(record(t, rec), testSync)
			require.NoError(t, err)
			require.True(t, keep)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestRunHashNumberAsJSONText(t *testing.T) {
	t.Parallel()

	out, _, err := runOne(t, ordersSchema, `[{"hash":["amount"]}]`, `{"amount":1e21}`)
	require.NoError(t, err)
	require.Equal(t, hexHash("s1e+21"), out["amount"])
}
