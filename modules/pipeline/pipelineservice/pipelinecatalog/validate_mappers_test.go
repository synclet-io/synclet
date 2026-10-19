package pipelinecatalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func validateUsers(t *testing.T, f *configureFixture, mappers, sample string) *ValidateMappersResult {
	t.Helper()

	params := ValidateMappersParams{
		SourceID:       f.conn.SourceID,
		ConnectionName: f.conn.Name,
		Stream: protocol.ConfiguredAirbyteStream{
			Stream:              protocol.AirbyteStream{Namespace: "public", Name: "users"},
			DestinationSyncMode: protocol.DestinationSyncModeAppendDedup,
			PrimaryKey:          [][]string{{"id"}},
			SelectedFields:      fields([]string{"name"}),
			Mappers:             json.RawMessage(mappers),
		},
	}
	if sample != "" {
		params.SampleRecord = json.RawMessage(sample)
	}

	result, err := NewValidateMappers(f.store).Execute(context.Background(), params)
	require.NoError(t, err)

	return result
}

func TestValidateMappers(t *testing.T) {
	t.Run("a bad expression returns that step's error and no schema", func(t *testing.T) {
		f := newConfigureFixture(t)

		result := validateUsers(t, f, `[{"rename":{"name":"full_name"}},{"set":{"label":"r.full_name +"}}]`, `{"id":1,"name":"Ann"}`)

		require.Len(t, result.Errors, 1)
		assert.Equal(t, 1, result.Errors[0].Step)
		assert.Equal(t, "label", result.Errors[0].Field)
		assert.Nil(t, result.OutputSchema)
		assert.Nil(t, result.SampleOutput)
	})

	t.Run("a valid list returns the output schema and the sample's output", func(t *testing.T) {
		f := newConfigureFixture(t)

		result := validateUsers(t, f,
			`[{"rename":{"id":"user_id"}},{"set":{"greeting":"'hi ' + r.name"}}]`,
			`{"id":7,"name":"Ann","updated_at":"2026-01-01","address":{"city":"Kyiv"}}`,
		)

		require.Empty(t, result.Errors)
		assert.JSONEq(t, `{
			"type": "object",
			"properties": {
				"user_id": {"type": "integer"},
				"name": {"type": "string"},
				"greeting": {"type": "string"}
			}
		}`, string(result.OutputSchema))
		assert.Equal(t, [][]string{{"user_id"}}, result.OutputPrimaryKey)
		assert.True(t, result.SampleKept)
		assert.Empty(t, result.SampleError)
		assert.JSONEq(t, `{"user_id":7,"name":"Ann","greeting":"hi Ann"}`, string(result.SampleOutput))
	})

	t.Run("a filter that drops the sample", func(t *testing.T) {
		f := newConfigureFixture(t)

		result := validateUsers(t, f, `[{"filter":"r.id > 10"}]`, `{"id":7,"name":"Ann"}`)

		require.Empty(t, result.Errors)
		assert.False(t, result.SampleKept)
		assert.Nil(t, result.SampleOutput)
	})

	t.Run("a step that fails on the sample", func(t *testing.T) {
		f := newConfigureFixture(t)

		result := validateUsers(t, f, `[{"cast":{"name":"integer"}}]`, `{"id":7,"name":"Ann"}`)

		require.Empty(t, result.Errors)
		assert.False(t, result.SampleKept)
		assert.Contains(t, result.SampleError, "mapper step 0")
	})

	t.Run("without a sample nothing is run", func(t *testing.T) {
		f := newConfigureFixture(t)

		result := validateUsers(t, f, `[{"rename":{"name":"full_name"}}]`, "")

		require.Empty(t, result.Errors)
		assert.NotNil(t, result.OutputSchema)
		assert.False(t, result.SampleKept)
		assert.Nil(t, result.SampleOutput)
	})

	t.Run("unknown stream is a validation error", func(t *testing.T) {
		f := newConfigureFixture(t)

		_, err := NewValidateMappers(f.store).Execute(context.Background(), ValidateMappersParams{
			SourceID: f.conn.SourceID,
			Stream:   protocol.ConfiguredAirbyteStream{Stream: protocol.AirbyteStream{Name: "nope"}, Mappers: json.RawMessage(`[]`)},
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})

	t.Run("mappers that are not a list of steps", func(t *testing.T) {
		f := newConfigureFixture(t)

		result := validateUsers(t, f, `{"rename":{}}`, "")

		require.Len(t, result.Errors, 1)
		assert.Equal(t, mapper.SelectionStep, result.Errors[0].Step)
	})
}
