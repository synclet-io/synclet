package mapper

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

const ordersSchema = `{
  "type": "object",
  "properties": {
    "id": {"type": "integer"},
    "cust_id": {"type": "integer"},
    "amount": {"type": ["null", "number"]},
    "email": {"type": "string"},
    "status": {"type": "string"},
    "code": {"type": "string"},
    "address": {"type": "object", "properties": {"city": {"type": "string"}, "zip": {"type": "string"}}},
    "items": {"type": "array", "items": {"type": "integer"}}
  }
}`

func TestCompileErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		steps   string
		issue   Issue
		partial bool
	}{
		{"rename source missing", `[{"rename":{"nope":"x"}}]`, Issue{0, "nope", `field "nope" does not exist`}, false},
		{"field renamed away used in expression", `[{"rename":{"amount":"total"}},{"set":{"x":"r.amount * 2.0"}}]`, Issue{1, "amount", `field "amount" was renamed by step 0`}, false},
		{"field renamed away used by hash", `[{"rename":{"email":"mail"}},{"hash":["email"]}]`, Issue{1, "email", `field "email" was renamed by step 0`}, false},
		{"field renamed away renamed again", `[{"rename":{"email":"mail"}},{"rename":{"email":"e"}}]`, Issue{1, "email", `field "email" was renamed by step 0`}, false},
		{"hash of missing field", `[{"hash":["nope"]}]`, Issue{0, "nope", `field "nope" does not exist`}, false},
		{"mask of missing field", `[{"mask":{"nope":{"keep_last":4}}}]`, Issue{0, "nope", `field "nope" does not exist`}, false},
		{"cast of missing field", `[{"cast":{"nope":"integer"}}]`, Issue{0, "nope", `field "nope" does not exist`}, false},
		{"rename onto existing field", `[{"rename":{"cust_id":"id"}}]`, Issue{0, "id", `field "id" already exists`}, false},
		{"two steps creating the same field", `[{"set":{"x":"1"}},{"set":{"x":"2"}}]`, Issue{1, "x", `field "x" already exists`}, false},
		{"rename target created by an earlier set", `[{"set":{"x":"1"}},{"rename":{"code":"x"}}]`, Issue{1, "x", `field "x" already exists`}, false},
		{"one rename mapping two fields to one name", `[{"rename":{"code":"x","status":"x"}}]`, Issue{0, "x", `field "x" already exists`}, false},
		{"empty field name", `[{"set":{"":"1"}}]`, Issue{0, "", `invalid field name ""`}, false},
		{"leading space", `[{"rename":{"code":" code"}}]`, Issue{0, " code", `invalid field name " code"`}, false},
		{"trailing space", `[{"set":{"x ":"1"}}]`, Issue{0, "x ", `invalid field name "x "`}, false},
		{"control character", `[{"set":{"a\tb":"1"}}]`, Issue{0, "a\tb", `invalid field name "a\tb"`}, false},
		{"name too long", `[{"set":{"` + strings.Repeat("a", 256) + `":"1"}}]`, Issue{0, strings.Repeat("a", 256), `invalid field name "` + strings.Repeat("a", 256) + `"`}, false},
		{"CEL type error", `[{"set":{"x":"r.amount + 'a'"}}]`, Issue{0, "x", "no matching overload"}, true},
		{"CEL unknown field", `[{"set":{"x":"r.nope"}}]`, Issue{0, "x", "undefined field 'nope'"}, true},
		{"CEL syntax error", `[{"set":{"x":"r.amount +"}}]`, Issue{0, "x", "Syntax error"}, true},
		{"CEL error in cast source", `[{"cast":{"x":{"to":"integer","source":"r.code.nope()"}}}]`, Issue{0, "x", "undeclared reference to 'nope'"}, true},
		{"filter not bool", `[{"filter":"r.amount"}]`, Issue{0, "", "filter must return bool, got double"}, false},
		{"filter CEL error", `[{"filter":"r.status == 1"}]`, Issue{0, "", "no matching overload"}, true},
		{"unknown cast target", `[{"cast":{"code":"money"}}]`, Issue{0, "code", `unknown cast target "money"`}, false},
		{"unknown cast format", `[{"cast":{"code":{"to":"timestamp","format":"QQ.yyyy"}}}]`, Issue{0, "code", `unknown cast format "QQ.yyyy"`}, false},
		{"unknown timezone", `[{"cast":{"code":{"to":"timestamp","timezone":"Mars/Base"}}}]`, Issue{0, "code", `unknown timezone "Mars/Base"`}, false},
		{"unknown cast on_error", `[{"cast":{"code":{"to":"integer","on_error":"retry"}}}]`, Issue{0, "code", `on_error "retry" is not allowed on cast`}, false},
		{"on_error on rename", `[{"rename":{"code":"c"},"on_error":"skip"}]`, Issue{0, "", `on_error "skip" is not allowed on rename`}, false},
		{"on_error null on filter", `[{"filter":"true","on_error":"null"}]`, Issue{0, "", `on_error "null" is not allowed on filter`}, false},
		{"unknown on_error on set", `[{"set":{"x":"1"},"on_error":"retry"}]`, Issue{0, "", `on_error "retry" is not allowed on set`}, false},
		{"negative mask", `[{"mask":{"code":{"keep_last":-1}}}]`, Issue{0, "code", "keep_first and keep_last must not be negative"}, false},
		{"empty step", `[{}]`, Issue{0, "", "step must hold exactly one mapper"}, false},
		{"two mappers in a step", `[{"set":{"x":"1"},"filter":"true"}]`, Issue{0, "", "step must hold exactly one mapper"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain, res := Compile(Input{Schema: json.RawMessage(ordersSchema), Steps: steps(t, tt.steps)})
			require.Nil(t, chain)
			require.Len(t, res.Errors, 1, "%+v", res.Errors)

			got := res.Errors[0]
			require.Equal(t, tt.issue.Step, got.Step)
			require.Equal(t, tt.issue.Field, got.Field)

			if tt.partial {
				require.Contains(t, got.Message, tt.issue.Message)
			} else {
				require.Equal(t, tt.issue.Message, got.Message)
			}
		})
	}
}

func TestCompilePrimaryKey(t *testing.T) {
	t.Parallel()

	t.Run("unselected primary key in a dedup mode", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []protocol.DestinationSyncMode{protocol.DestinationSyncModeAppendDedup, protocol.DestinationSyncModeUpdate, protocol.DestinationSyncModeSoftDelete} {
			chain, res := Compile(Input{
				Schema:              json.RawMessage(ordersSchema),
				SelectedFields:      []protocol.SelectedField{{FieldPath: []string{"email"}}},
				PrimaryKey:          [][]string{{"id"}},
				DestinationSyncMode: mode,
			})
			require.Nil(t, chain)
			require.Equal(t, []Issue{{SelectionStep, "id", `primary key field "id" is not selected`}}, res.Errors)
		}
	})

	t.Run("unselected primary key outside dedup", func(t *testing.T) {
		t.Parallel()

		mustCompile(t, Input{
			Schema:              json.RawMessage(ordersSchema),
			SelectedFields:      []protocol.SelectedField{{FieldPath: []string{"email"}}},
			PrimaryKey:          [][]string{{"id"}},
			DestinationSyncMode: protocol.DestinationSyncModeAppend,
		})
	})

	t.Run("nested primary key covered by its parent", func(t *testing.T) {
		t.Parallel()

		mustCompile(t, Input{
			Schema:              json.RawMessage(ordersSchema),
			SelectedFields:      []protocol.SelectedField{{FieldPath: []string{"address"}}},
			PrimaryKey:          [][]string{{"address", "zip"}},
			DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
		})
	})

	t.Run("rename rewrites the primary key", func(t *testing.T) {
		t.Parallel()

		_, res := mustCompile(t, Input{
			Schema:              json.RawMessage(ordersSchema),
			Steps:               steps(t, `[{"rename":{"id":"key"}},{"rename":{"key":"order_key"}}]`),
			PrimaryKey:          [][]string{{"id"}, {"address", "zip"}},
			DestinationSyncMode: protocol.DestinationSyncModeSoftDelete,
			SelectedFields:      []protocol.SelectedField{{FieldPath: []string{"address"}}},
		})
		require.Equal(t, [][]string{{"order_key"}, {"address", "zip"}}, res.PrimaryKey)
		require.Empty(t, res.Warnings)
	})
}

func TestCompileFilterWarning(t *testing.T) {
	t.Parallel()

	const cdcSchema = `{"type":"object","properties":{"id":{"type":"integer"},"_ab_cdc_lsn":{"type":"integer"}}}`

	tests := []struct {
		name   string
		schema string
		mode   protocol.DestinationSyncMode
		warn   bool
	}{
		{"append_dedup", ordersSchema, protocol.DestinationSyncModeAppendDedup, true},
		{"update", ordersSchema, protocol.DestinationSyncModeUpdate, true},
		{"soft_delete", ordersSchema, protocol.DestinationSyncModeSoftDelete, true},
		{"cdc", cdcSchema, protocol.DestinationSyncModeAppend, true},
		{"append", ordersSchema, protocol.DestinationSyncModeAppend, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, res := mustCompile(t, Input{
				Schema:              json.RawMessage(tt.schema),
				Steps:               steps(t, `[{"set":{"x":"1"}},{"filter":"r.id > 0"}]`),
				DestinationSyncMode: tt.mode,
			})

			if tt.warn {
				require.Equal(t, []Issue{{1, "", "rows filtered out later stay in the destination"}}, res.Warnings)
			} else {
				require.Empty(t, res.Warnings)
			}
		})
	}
}

func TestCompileOutputSchema(t *testing.T) {
	t.Parallel()

	_, res := mustCompile(t, Input{
		Schema: json.RawMessage(`{"type":"object","required":["cust_id"],"properties":{
			"cust_id":{"type":"integer"},
			"amount":{"type":"string"},
			"email":{"type":"string"},
			"card":{"type":"string"},
			"created":{"type":"string"},
			"tags":{"type":"string"}
		}}`),
		Steps: steps(t, `[
			{"rename":{"cust_id":"customer_id"}},
			{"cast":{"amount":"number","created":{"to":"timestamp","format":"unix"},"tags":"array"}},
			{"set":{"currency":"'USD'","total_cents":"int(r.amount * 100.0)","synced_at":"sync.started_at","big":"r.amount > 10.0","label":"coalesce(r.email, 'none')"}},
			{"hash":["email"]},
			{"mask":{"card":{"keep_last":4}}},
			{"filter":"r.total_cents > 0"}
		]`),
	})

	require.JSONEq(t, `{"type":"object","required":["customer_id"],"properties":{
		"customer_id":{"type":"integer"},
		"amount":{"type":"number"},
		"email":{"type":"string"},
		"card":{"type":"string"},
		"created":{"type":"string","format":"date-time","airbyte_type":"timestamp_with_timezone"},
		"tags":{"type":"array"},
		"currency":{"type":"string"},
		"total_cents":{"type":"integer"},
		"synced_at":{"type":"string","format":"date-time","airbyte_type":"timestamp_with_timezone"},
		"big":{"type":"boolean"},
		"label":{}
	}}`, string(res.Schema))
}

func TestCompileSetOntoRenamedField(t *testing.T) {
	t.Parallel()

	chain, res := mustCompile(t, Input{
		Schema: json.RawMessage(ordersSchema),
		Steps:  steps(t, `[{"rename":{"code":"x"}},{"set":{"x":"r.x + '!'"}}]`),
	})
	require.Contains(t, string(res.Schema), `"x":{"type":"string"}`)

	out, keep, err := chain.Run(record(t, `{"code":"abc"}`), testSync)
	require.NoError(t, err)
	require.True(t, keep)
	require.Equal(t, map[string]any{"x": "abc!"}, out)
}

func TestCompileHashMaskKeepNullable(t *testing.T) {
	t.Parallel()

	_, res := mustCompile(t, Input{
		Schema: json.RawMessage(`{"type":"object","properties":{
			"a":{"type":["null","string"]},
			"b":{"type":["integer","null"]},
			"c":{"type":"string"}
		}}`),
		Steps: steps(t, `[{"hash":["a","c"]},{"mask":{"b":{"keep_last":2}}}]`),
	})
	require.JSONEq(t, `{"type":"object","properties":{
		"a":{"type":["null","string"]},
		"b":{"type":["null","string"]},
		"c":{"type":"string"}
	}}`, string(res.Schema))
}

func TestCompileNullableOutput(t *testing.T) {
	t.Parallel()

	_, res := mustCompile(t, Input{
		Schema: json.RawMessage(`{"type":"object","properties":{
			"n":{"type":["null","string"]},
			"s":{"type":"string"},
			"t":{"type":"string"},
			"h":{"type":"string"},
			"m":{"type":"string"},
			"o":{"type":"string"}
		}}`),
		Steps: steps(t, `[
			{"cast":{"n":"integer","s":"integer","t":{"to":"timestamp","on_error":"null"}}},
			{"set":{"x":"r.s + 1","y":"r.o"},"on_error":"null"},
			{"set":{"z":"1"}},
			{"hash":["h"],"on_error":"null"},
			{"mask":{"m":{"keep_last":1}},"on_error":"null"},
			{"cast":{"o":"object"},"on_error":"null"}
		]`),
	})
	require.JSONEq(t, `{"type":"object","properties":{
		"n":{"type":["null","integer"]},
		"s":{"type":"integer"},
		"t":{"type":["null","string"],"format":"date-time","airbyte_type":"timestamp_with_timezone"},
		"x":{"type":["null","integer"]},
		"y":{"type":["null","string"]},
		"z":{"type":"integer"},
		"h":{"type":["null","string"]},
		"m":{"type":["null","string"]},
		"o":{"type":["null","object"]}
	}}`, string(res.Schema))
}

func TestFieldSelectionRenameKeepsSubtree(t *testing.T) {
	t.Parallel()

	chain, res := mustCompile(t, Input{
		Schema: json.RawMessage(ordersSchema),
		Steps:  steps(t, `[{"rename":{"address":"addr"}},{"rename":{"addr":"location"}},{"rename":{"email":"mail"}}]`),
		SelectedFields: []protocol.SelectedField{
			{FieldPath: []string{"id"}},
			{FieldPath: []string{"address", "city"}},
		},
	})
	require.JSONEq(t, `{"type":"object","properties":{
		"id":{"type":"integer"},
		"location":{"type":"object","properties":{"city":{"type":"string"}}},
		"mail":{"type":"string"}
	}}`, string(res.Schema))

	out, _, err := chain.Run(record(t, `{"id":1,"email":"e","address":{"city":"K","zip":"Z"}}`), testSync)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"id": int64(1), "mail": "e", "location": map[string]any{"city": "K"}}, out)
}

func TestCompileNullableExpressionResult(t *testing.T) {
	t.Parallel()

	chain, res := mustCompile(t, Input{
		Schema: json.RawMessage(`{"type":"object","properties":{
			"opt":{"type":["null","string"]},
			"nest":{"type":"object","properties":{"v":{"type":["null","integer"]}}},
			"s":{"type":"string"}
		}}`),
		Steps: steps(t, `[
			{"set":{"x":"r.opt","n":"r.nest.v","plain":"r.s","cmp":"r.opt == 'a'"}},
			{"cast":{"y":{"to":"integer","source":"r.opt"}}}
		]`),
	})
	require.JSONEq(t, `{"type":"object","properties":{
		"opt":{"type":["null","string"]},
		"nest":{"type":"object","properties":{"v":{"type":["null","integer"]}}},
		"s":{"type":"string"},
		"x":{"type":["null","string"]},
		"n":{"type":["null","integer"]},
		"plain":{"type":"string"},
		"cmp":{"type":"boolean"},
		"y":{"type":["null","integer"]}
	}}`, string(res.Schema))

	out, _, err := chain.Run(record(t, `{"opt":null,"nest":{"v":null},"s":"q"}`), testSync)
	require.NoError(t, err)
	require.Nil(t, out["x"])
	require.Nil(t, out["y"])
}

func TestCompileSetAfterRenameAway(t *testing.T) {
	t.Parallel()

	chain, _ := mustCompile(t, Input{
		Schema: json.RawMessage(ordersSchema),
		Steps:  steps(t, `[{"set":{"x":"1"}},{"rename":{"x":"y"}},{"set":{"x":"2"}}]`),
	})

	out, _, err := chain.Run(record(t, `{}`), testSync)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"y": int64(1), "x": int64(2)}, out)
}

func TestCompileRenameSwapAndChain(t *testing.T) {
	t.Parallel()

	const schema = `{"type":"object","required":["a"],"properties":{"a":{"type":"string"},"b":{"type":"integer"}}}`

	tests := []struct {
		name     string
		steps    string
		pk       [][]string
		required string
		props    string
	}{
		{
			"swap",
			`[{"rename":{"a":"b","b":"a"}}]`,
			[][]string{{"b"}},
			`["b"]`,
			`"a":{"type":"integer"},"b":{"type":"string"}`,
		},
		{
			"chain",
			`[{"rename":{"a":"b","b":"x"}}]`,
			[][]string{{"b"}},
			`["b"]`,
			`"b":{"type":"string"},"x":{"type":"integer"}`,
		},
		{
			"swap, then later steps read and replace the swapped names",
			`[{"rename":{"a":"b","b":"a"}},{"set":{"c":"r.a + 1","d":"r.b + '!'"}},{"set":{"a":"r.a * 2"}}]`,
			[][]string{{"b"}},
			`["b"]`,
			`"a":{"type":"integer"},"b":{"type":"string"},"c":{"type":"integer"},"d":{"type":"string"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, res := mustCompile(t, Input{
				Schema:              json.RawMessage(schema),
				Steps:               steps(t, tt.steps),
				PrimaryKey:          [][]string{{"a"}},
				DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
				SelectedFields:      []protocol.SelectedField{{FieldPath: []string{"a"}}, {FieldPath: []string{"b"}}},
			})
			require.Equal(t, tt.pk, res.PrimaryKey)
			require.JSONEq(t, `{"type":"object","required":`+tt.required+`,"properties":{`+tt.props+`}}`, string(res.Schema))
		})
	}
}
