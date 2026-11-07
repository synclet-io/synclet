package pipelinesecrets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

const arraySpec = `{"connectionSpecification":{"properties":{
	"account_id":{"type":"string"},
	"graph_api_headers":{"type":"array","items":{"type":"object","required":["name","value"],"properties":{
		"name":{"type":"string"},
		"value":{"type":"string","airbyte_secret":true}
	}}},
	"tokens":{"type":"array","items":{"type":"string","airbyte_secret":true}},
	"auth":{"type":"object","oneOf":[
		{"properties":{"mode":{"const":"headers"},"extra":{"type":"array","items":{"type":"object","properties":{
			"key":{"type":"string","airbyte_secret":true},
			"nested":{"type":"array","items":{"anyOf":[{"properties":{"pin":{"type":"string","airbyte_secret":true}}}]}}
		}}}}}
	]}
}}}`

func TestExtractSecretPaths_ArrayItems(t *testing.T) {
	paths, err := ExtractSecretPaths(arraySpec)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"auth.extra.*.key",
		"auth.extra.*.nested.*.pin",
		"graph_api_headers.*.value",
		"tokens.*",
	}, paths)
}

func TestExpandSecretPaths(t *testing.T) {
	var cfg map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{
		"graph_api_headers":[{"name":"a","value":"x"},{"name":"b"}],
		"tokens":["t1","t2"],
		"auth":{"extra":[{"key":"k","nested":[{"pin":"1"}]}]}
	}`), &cfg))

	paths, err := ExtractSecretPaths(arraySpec)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"auth.extra.0.key",
		"auth.extra.0.nested.0.pin",
		"graph_api_headers.0.value",
		"tokens.0",
		"tokens.1",
	}, ExpandSecretPaths(cfg, paths))

	val, ok := GetNestedField(cfg, "auth.extra.0.nested.0.pin")
	require.True(t, ok)
	assert.Equal(t, "1", val)

	SetNestedField(cfg, "tokens.1", "replaced")
	assert.Equal(t, []any{"t1", "replaced"}, cfg["tokens"])
}

func TestEncryptConfigSecrets_ArrayItems(t *testing.T) {
	secrets := newFakeSecrets()

	got, err := EncryptConfigSecrets(context.Background(), secrets, "source", uuid.New(),
		`{"account_id":"1","graph_api_headers":[{"name":"a","value":"va"},{"name":"b","value":"vb"}],"tokens":["t1"]}`, arraySpec)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal([]byte(got), &cfg))

	headers, ok := cfg["graph_api_headers"].([]any)
	require.True(t, ok)
	require.Len(t, headers, 2)

	for i, want := range []string{"va", "vb"} {
		header, _ := headers[i].(map[string]any)
		ref, _ := header["value"].(string)
		require.True(t, secretutil.IsSecretRef(ref), "header %d value is %q", i, ref)
		assert.Equal(t, want, secrets.stored[ref])
	}

	first, _ := headers[0].(map[string]any)
	assert.Equal(t, "a", first["name"])

	tokens, _ := cfg["tokens"].([]any)
	require.Len(t, tokens, 1)

	tokenRef, _ := tokens[0].(string)
	assert.Equal(t, "t1", secrets.stored[tokenRef])
}

func TestMaskConfigSecrets_ArrayItems(t *testing.T) {
	got, err := MaskConfigSecrets(`{"graph_api_headers":[{"name":"a","value":"` + storedRef + `"}],"tokens":["` + otherRef + `","plain"],"grid":[["` + storedRef + `"]]}`)
	require.NoError(t, err)

	assert.JSONEq(t, `{"graph_api_headers":[{"name":"a","value":"********"}],"tokens":["********","plain"],"grid":[["********"]]}`, got)
}

func TestExtractSecretRefs_ArrayOfStrings(t *testing.T) {
	refs, err := ExtractSecretRefs(`{"tokens":["` + storedRef + `"],"h":[{"v":"` + otherRef + `"}]}`)
	require.NoError(t, err)
	assert.Len(t, refs, 2)
}
