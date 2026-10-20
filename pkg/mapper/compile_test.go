package mapper

import (
	"encoding/json"
	"slices"
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
		{"drop of missing field", `[{"drop":["nope"]}]`, Issue{0, "nope", `field "nope" does not exist`}, false},
		{"drop of field renamed away", `[{"rename":{"email":"mail"}},{"drop":["email"]}]`, Issue{1, "email", `field "email" was renamed by step 0`}, false},
		{"field listed twice in a drop", `[{"drop":["email","email"]}]`, Issue{0, "email", `field "email" is listed twice`}, false},
		{"dropped field used in expression", `[{"drop":["amount"]},{"set":{"x":"r.amount * 2.0"}}]`, Issue{1, "amount", `field "amount" was dropped by step 0`}, false},
		{"dropped field used in filter", `[{"drop":["status"]},{"filter":"r.status == 'x'"}]`, Issue{1, "status", `field "status" was dropped by step 0`}, false},
		{"dropped field used by hash", `[{"drop":["email"]},{"hash":["email"]}]`, Issue{1, "email", `field "email" was dropped by step 0`}, false},
		{"dropped field dropped again", `[{"drop":["email"]},{"drop":["email"]}]`, Issue{1, "email", `field "email" was dropped by step 0`}, false},
		{"on_error on drop", `[{"drop":["code"],"on_error":"skip"}]`, Issue{0, "", `on_error "skip" is not allowed on drop`}, false},
		{"catch arguments of different types", `[{"set":{"x":"catch(int(r.code), 'n/a')"}}]`, Issue{0, "x", "no matching overload for 'catch'"}, true},
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

	t.Run("drop of a primary key in a keyed mode", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []protocol.DestinationSyncMode{protocol.DestinationSyncModeAppendDedup, protocol.DestinationSyncModeUpdate, protocol.DestinationSyncModeSoftDelete} {
			chain, res := Compile(Input{
				Schema:              json.RawMessage(ordersSchema),
				Steps:               steps(t, `[{"rename":{"address":"addr"}},{"drop":["addr"]}]`),
				PrimaryKey:          [][]string{{"address", "zip"}},
				DestinationSyncMode: mode,
			})
			require.Nil(t, chain)
			require.Equal(t, []Issue{{1, "addr", `primary key field "addr.zip" cannot be dropped in ` + string(mode) + ` mode`}}, res.Errors)
		}
	})

	t.Run("drop of a primary key outside keyed modes removes it from the key", func(t *testing.T) {
		t.Parallel()

		_, res := mustCompile(t, Input{
			Schema:              json.RawMessage(ordersSchema),
			Steps:               steps(t, `[{"drop":["id"]}]`),
			PrimaryKey:          [][]string{{"id"}, {"address", "zip"}},
			DestinationSyncMode: protocol.DestinationSyncModeAppend,
		})
		require.Equal(t, [][]string{{"address", "zip"}}, res.PrimaryKey)
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
		"label":{"type":"string"}
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

func TestCompileDateStringConversion(t *testing.T) {
	t.Parallel()

	const schema = `{
  "type": "object",
  "properties": {
    "tstamp": {"type": "string", "format": "date-time", "airbyte_type": "timestamp_without_timezone"},
    "day": {"type": ["null", "string"], "format": "date"},
    "at": {"type": "string", "format": "time"},
    "meta": {"type": "object", "properties": {"seen": {"type": "string", "format": "date-time"}}},
    "qty": {"type": "string"}
  }
}`

	tests := []struct {
		name  string
		steps string
		issue *Issue
	}{
		{"int of a date-time", `[{"set":{"epoch":"int(r.tstamp)"}}]`, &Issue{0, "epoch", `"tstamp" is a date-time string, int() cannot read it; use int(parseTimestamp(r.tstamp, "iso8601")) for epoch seconds`}},
		{"double of a nullable date", `[{"set":{"d":"double(r.day)"}}]`, &Issue{0, "d", `"day" is a date string, double() cannot read it; use double(parseTimestamp(r.day, "iso8601")) for epoch seconds`}},
		{"uint of a nested date-time", `[{"set":{"s":"uint(r.meta.seen)"}}]`, &Issue{0, "s", `"meta.seen" is a date-time string, uint() cannot read it; use uint(parseTimestamp(r.meta.seen, "iso8601")) for epoch seconds`}},
		{"int of a time", `[{"set":{"t":"int(r.at)"}}]`, &Issue{0, "t", `"at" is a time string, int() cannot read it`}},
		{"inside a filter", `[{"filter":"int(r.tstamp) > 0"}]`, &Issue{0, "", `"tstamp" is a date-time string, int() cannot read it; use int(parseTimestamp(r.tstamp, "iso8601")) for epoch seconds`}},
		{"int of a plain string", `[{"set":{"n":"int(r.qty)"}}]`, nil},
		{"int of a parsed date-time", `[{"set":{"epoch":"int(parseTimestamp(r.tstamp, \"iso8601\"))"}}]`, nil},
		{"string of a date-time", `[{"set":{"s":"string(r.tstamp)"}}]`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, res := Compile(Input{Schema: json.RawMessage(schema), Steps: steps(t, tt.steps)})
			if tt.issue == nil {
				require.Empty(t, res.Errors)

				return
			}

			require.Equal(t, []Issue{*tt.issue}, res.Errors)
		})
	}
}

func TestCompileDrop(t *testing.T) {
	t.Parallel()

	const schema = `{"type":"object","required":["id","amount"],"properties":{
		"id":{"type":"integer"},
		"amount":{"type":"number"},
		"status":{"type":"string"}
	}}`

	tests := []struct {
		name     string
		steps    string
		selected []protocol.SelectedField
		schema   string
		want     map[string]any
	}{
		{
			"no selection",
			`[{"set":{"total":"r.amount * 2.0"}},{"drop":["amount","status"]}]`,
			nil,
			`{"type":"object","required":["id"],"properties":{"id":{"type":"integer"},"total":{"type":"number"}}}`,
			map[string]any{"id": int64(1), "total": 3.0},
		},
		{
			"dropped field selected",
			`[{"set":{"total":"r.amount * 2.0"}},{"drop":["amount"]}]`,
			[]protocol.SelectedField{{FieldPath: []string{"id"}}, {FieldPath: []string{"amount"}}},
			`{"type":"object","required":["id"],"properties":{"id":{"type":"integer"},"total":{"type":"number"}}}`,
			map[string]any{"id": int64(1), "total": 3.0},
		},
		{
			"dropped field written by an earlier step",
			`[{"set":{"tmp":"r.amount * 2.0"}},{"set":{"total":"r.tmp + 1.0"}},{"drop":["tmp"]}]`,
			[]protocol.SelectedField{{FieldPath: []string{"id"}}, {FieldPath: []string{"amount"}}},
			`{"type":"object","required":["id","amount"],"properties":{"id":{"type":"integer"},"amount":{"type":"number"},"total":{"type":"number"}}}`,
			map[string]any{"id": int64(1), "amount": 1.5, "total": 4.0},
		},
		{
			"dropped renamed field",
			`[{"rename":{"status":"state"}},{"drop":["state"]}]`,
			nil,
			`{"type":"object","required":["id","amount"],"properties":{"id":{"type":"integer"},"amount":{"type":"number"}}}`,
			map[string]any{"id": int64(1), "amount": 1.5},
		},
		{
			"set recreates a dropped name",
			`[{"drop":["status"]},{"set":{"status":"'new'"}}]`,
			[]protocol.SelectedField{{FieldPath: []string{"id"}}, {FieldPath: []string{"status"}}},
			`{"type":"object","required":["id"],"properties":{"id":{"type":"integer"},"status":{"type":"string"}}}`,
			map[string]any{"id": int64(1), "status": "new"},
		},
		{
			"drop every field",
			`[{"drop":["id","amount","status"]}]`,
			nil,
			`{"type":"object","required":[],"properties":{}}`,
			map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain, res := mustCompile(t, Input{Schema: json.RawMessage(schema), Steps: steps(t, tt.steps), SelectedFields: tt.selected})
			require.JSONEq(t, tt.schema, string(res.Schema))

			out, keep, err := chain.Run(record(t, `{"id":1,"amount":1.5,"status":"old"}`), testSync)
			require.NoError(t, err)
			require.True(t, keep)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestCompileUnselectedFieldWarning(t *testing.T) {
	t.Parallel()

	selectID := []protocol.SelectedField{{FieldPath: []string{"id"}}}

	tests := []struct {
		name     string
		steps    string
		selected []protocol.SelectedField
		want     []Issue
	}{
		{"set reads an unselected field", `[{"set":{"x":"r.status + '!'"}}]`, selectID, []Issue{{0, "status", unselectedMessage("status")}}},
		{"filter reads an unselected field", `[{"set":{"x":"1"}},{"filter":"r.status == 'a'"}]`, selectID, []Issue{{1, "status", unselectedMessage("status")}}},
		{"cast source reads an unselected field", `[{"cast":{"n":{"to":"integer","source":"r.code"}}}]`, selectID, []Issue{{0, "code", unselectedMessage("code")}}},
		{"one warning per field and step", `[{"set":{"x":"r.status","y":"r.status + r.code"}}]`, selectID, []Issue{{0, "status", unselectedMessage("status")}, {0, "code", unselectedMessage("code")}}},
		{"renamed unselected field names its source", `[{"rename":{"status":"state"}},{"set":{"x":"r.state"}}]`, selectID, []Issue{{1, "status", unselectedMessage("status")}}},
		{"selected field", `[{"set":{"x":"r.status"}}]`, []protocol.SelectedField{{FieldPath: []string{"id"}}, {FieldPath: []string{"status"}}}, nil},
		{"nested selection covers its parent", `[{"set":{"x":"r.address.city"}}]`, []protocol.SelectedField{{FieldPath: []string{"address", "city"}}}, nil},
		{"field created by an earlier step", `[{"set":{"x":"1"}},{"set":{"y":"r.x + 1"}}]`, selectID, nil},
		{"field recomputed by an earlier step", `[{"set":{"status":"'a'"}},{"set":{"y":"r.status"}}]`, selectID, nil},
		{"no selection", `[{"set":{"x":"r.status"}}]`, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, res := mustCompile(t, Input{Schema: json.RawMessage(ordersSchema), Steps: steps(t, tt.steps), SelectedFields: tt.selected})
			require.Equal(t, tt.want, res.Warnings)
		})
	}
}

func unselectedMessage(field string) string {
	return `field "` + field + `" is not selected; sources that honour field selection will not send it — select it and drop it after use`
}

func TestCompileCatchType(t *testing.T) {
	t.Parallel()

	_, res := mustCompile(t, Input{
		Schema: json.RawMessage(`{"type":"object","properties":{"s":{"type":"string"}}}`),
		Steps:  steps(t, `[{"set":{"x":"catch(int(r.s), 0)"}}]`),
	})
	require.JSONEq(t, `{"type":"object","properties":{"s":{"type":"string"},"x":{"type":"integer"}}}`, string(res.Schema))
}

func TestCompileCursorDrop(t *testing.T) {
	t.Parallel()

	cursorMessage := func(field string) string {
		return `cursor field "` + field + `" cannot be dropped in incremental mode; choose another cursor in the stream settings or keep the field`
	}

	tests := []struct {
		name   string
		steps  string
		cursor []string
		mode   protocol.SyncMode
		issue  *Issue
		want   []string
	}{
		{"incremental drop of the cursor", `[{"drop":["status"]}]`, []string{"status"}, protocol.SyncModeIncremental, &Issue{0, "status", cursorMessage("status")}, nil},
		{"incremental drop of the renamed cursor", `[{"rename":{"status":"state"}},{"drop":["state"]}]`, []string{"status"}, protocol.SyncModeIncremental, &Issue{1, "state", cursorMessage("state")}, nil},
		{"incremental drop of a nested cursor's parent", `[{"drop":["address"]}]`, []string{"address", "zip"}, protocol.SyncModeIncremental, &Issue{0, "address", cursorMessage("address.zip")}, nil},
		{"full refresh drop of the cursor", `[{"drop":["status"]}]`, []string{"status"}, protocol.SyncModeFullRefresh, nil, nil},
		{"incremental drop of another field", `[{"drop":["code"]}]`, []string{"status"}, protocol.SyncModeIncremental, nil, []string{"status"}},
		{"rename moves the cursor", `[{"rename":{"address":"addr"}}]`, []string{"address", "zip"}, protocol.SyncModeIncremental, nil, []string{"addr", "zip"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := Input{
				Schema:      json.RawMessage(ordersSchema),
				Steps:       steps(t, tt.steps),
				CursorField: slices.Clone(tt.cursor),
				SyncMode:    tt.mode,
			}

			if tt.issue != nil {
				chain, res := Compile(in)
				require.Nil(t, chain)
				require.Equal(t, []Issue{*tt.issue}, res.Errors)

				return
			}

			_, res := mustCompile(t, in)
			require.Equal(t, tt.want, res.CursorField)
		})
	}
}
