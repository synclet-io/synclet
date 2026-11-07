package pipelinesources_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

const facebookMarketingSpec = `{"connectionSpecification":{"type":"object","required":["account_ids","access_token"],"properties":{
	"account_ids":{"type":"array","items":{"type":"string"}},
	"access_token":{"type":"string","airbyte_secret":true},
	"graph_api_headers":{"type":"array","items":{"type":"object","required":["name","value"],"properties":{
		"name":{"type":"string"},
		"value":{"type":"string","airbyte_secret":true}
	}}}
}}}`

func headerValueRefs(t *testing.T, config string) map[string]string {
	t.Helper()

	headers, ok := configMap(t, config)["graph_api_headers"].([]any)
	require.True(t, ok, "graph_api_headers is not an array")

	refs := map[string]string{}

	for _, item := range headers {
		header, ok := item.(map[string]any)
		require.True(t, ok, "header is not an object: %v", item)

		name, _ := header["name"].(string)
		ref, _ := header["value"].(string)
		require.True(t, secretutil.IsSecretRef(ref), "header %q value is %q", name, ref)

		refs[name] = ref
	}

	return refs
}

func TestFacebookMarketingGraphAPIHeaders(t *testing.T) {
	ws := uuid.New()
	store := newSrcStore()
	conn := store.addConnector(ws, facebookMarketingSpec)
	secrets := newFakeSecrets()

	created, err := pipelinesources.NewCreateSource(store, secrets, &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.CreateSourceParams{
		WorkspaceID:        ws,
		Name:               "fb",
		ManagedConnectorID: conn.ID,
		Config: json.RawMessage(`{"account_ids":["1"],"access_token":"tok","graph_api_headers":[
			{"name":"X-Business-Use-Case","value":"header-one"},
			{"name":"X-App-Usage","value":"header-two"}
		]}`),
	})
	require.NoError(t, err)

	refs := headerValueRefs(t, created.Config)
	require.Len(t, refs, 2)
	assert.Equal(t, "header-one", secrets.values[refs["X-Business-Use-Case"]])
	assert.Equal(t, "header-two", secrets.values[refs["X-App-Usage"]])

	masked, err := pipelinesecrets.MaskConfigSecrets(created.Config)
	require.NoError(t, err)
	assert.NotContains(t, masked, "header-one")

	var edited map[string]any
	require.NoError(t, json.Unmarshal([]byte(masked), &edited))
	edited["account_ids"] = []any{"1", "2"}

	editedJSON, err := json.Marshal(edited)
	require.NoError(t, err)

	updated, err := pipelinesources.NewUpdateSource(store, secrets, &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.UpdateSourceParams{
		ID:          created.ID,
		WorkspaceID: ws,
		Config:      rawJSON(string(editedJSON)),
	})
	require.NoError(t, err)

	assert.Equal(t, refs, headerValueRefs(t, updated.Config))
	assert.Equal(t, []any{"1", "2"}, configMap(t, updated.Config)["account_ids"])
	assert.Empty(t, secrets.deletedRefs)
}
