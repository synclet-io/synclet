package pipelinerepositories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
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

	fetcher := NewRegistryFetcher(pipelineservice.Config{RegistryFetchTimeout: 60 * time.Second})
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Contains(t, connectors[0].Spec, `"api_key"`)
	assert.Contains(t, connectors[0].Spec, `"type":`)
	assert.Equal(t, pipelineservice.ConnectorTypeSource, connectors[0].ConnectorType)
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

	fetcher := NewRegistryFetcher(pipelineservice.Config{RegistryFetchTimeout: 60 * time.Second})
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

	fetcher := NewRegistryFetcher(pipelineservice.Config{RegistryFetchTimeout: 60 * time.Second})
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

	fetcher := NewRegistryFetcher(pipelineservice.Config{RegistryFetchTimeout: 60 * time.Second})
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Contains(t, connectors[0].Spec, `"host"`)
	assert.Equal(t, pipelineservice.ConnectorTypeDestination, connectors[0].ConnectorType)
}

func fetchSingleSource(t *testing.T, entry string) ConnectorData {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sources": [` + entry + `], "destinations": []}`))
	}))
	defer server.Close()

	fetcher := NewRegistryFetcher(pipelineservice.Config{RegistryFetchTimeout: 60 * time.Second})
	connectors, err := fetcher.Fetch(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.Len(t, connectors, 1)

	return connectors[0]
}

func TestRegistryFetcherTopLevelBreakingChanges(t *testing.T) {
	connector := fetchSingleSource(t, `{
		"name": "test-source",
		"dockerRepository": "synclet/source-test",
		"dockerImageTag": "3.0.0",
		"breakingChanges": {
			"2.0.0": {"message": "Renamed streams", "migration_documentation_url": "https://docs.example.com/2", "upgrade_deadline": "2026-01-01"},
			"3.0.0": {"message": "Dropped v1 auth"}
		}
	}`)

	assert.Equal(t, map[string]pipelineservice.BreakingChange{
		"2.0.0": {Message: "Renamed streams", MigrationDocumentationURL: "https://docs.example.com/2", UpgradeDeadline: "2026-01-01"},
		"3.0.0": {Message: "Dropped v1 auth"},
	}, connector.Metadata.BreakingChanges)
}

func TestRegistryFetcherMergesBreakingChangesReleasesWin(t *testing.T) {
	connector := fetchSingleSource(t, `{
		"name": "test-source",
		"dockerRepository": "synclet/source-test",
		"dockerImageTag": "3.0.0",
		"breakingChanges": {
			"1.0.0": {"message": "top-level only"},
			"2.0.0": {"message": "top-level loses"}
		},
		"releases": {
			"breakingChanges": {
				"2.0.0": {"message": "releases wins"},
				"3.0.0": {"message": "releases only"}
			}
		}
	}`)

	assert.Equal(t, map[string]pipelineservice.BreakingChange{
		"1.0.0": {Message: "top-level only"},
		"2.0.0": {Message: "releases wins"},
		"3.0.0": {Message: "releases only"},
	}, connector.Metadata.BreakingChanges)
}
