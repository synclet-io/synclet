package pipelinerepositories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryFetcherSpecParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sources": [
				{
					"name": "test-source",
					"dockerRepository": "airbyte/source-test",
					"dockerImageTag": "1.0.0",
					"documentationUrl": "https://docs.example.com",
					"releaseStage": "generally_available",
					"icon": "test.svg",
					"tombstone": false,
					"spec": {
						"connectionSpecification": {"type": "object", "properties": {"api_key": {"type": "string"}}}
					}
				}
			],
			"destinations": []
		}`))
	}))
	defer server.Close()

	fetcher := NewRegistryFetcher()
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Contains(t, connectors[0].Spec, `"api_key"`)
	assert.Contains(t, connectors[0].Spec, `"type":`)
	assert.Equal(t, "source", connectors[0].ConnectorType)
}

func TestRegistryFetcherNoSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sources": [
				{
					"name": "no-spec-source",
					"dockerRepository": "airbyte/source-nospec",
					"dockerImageTag": "1.0.0",
					"tombstone": false
				}
			],
			"destinations": []
		}`))
	}))
	defer server.Close()

	fetcher := NewRegistryFetcher()
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Empty(t, connectors[0].Spec)
}

func TestRegistryFetcherSpecNilConnectionSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sources": [
				{
					"name": "nil-spec-source",
					"dockerRepository": "airbyte/source-nilspec",
					"dockerImageTag": "1.0.0",
					"tombstone": false,
					"spec": {}
				}
			],
			"destinations": []
		}`))
	}))
	defer server.Close()

	fetcher := NewRegistryFetcher()
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Empty(t, connectors[0].Spec)
}

func TestRegistryFetcherDestinationSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sources": [],
			"destinations": [
				{
					"name": "test-dest",
					"dockerRepository": "airbyte/destination-test",
					"dockerImageTag": "2.0.0",
					"tombstone": false,
					"spec": {
						"connectionSpecification": {"type": "object", "properties": {"host": {"type": "string"}}}
					}
				}
			]
		}`))
	}))
	defer server.Close()

	fetcher := NewRegistryFetcher()
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Contains(t, connectors[0].Spec, `"host"`)
	assert.Equal(t, "destination", connectors[0].ConnectorType)
}
