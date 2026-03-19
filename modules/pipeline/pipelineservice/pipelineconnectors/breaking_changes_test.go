package pipelineconnectors

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestDetectBreakingChanges(t *testing.T) {
	metadata := &pipelineservice.RepositoryConnectorMetadata{
		BreakingChanges: map[string]pipelineservice.BreakingChange{
			"1.0.0": {Message: "Initial release"},
			"2.0.0": {
				Message:                   "Removed legacy auth",
				MigrationDocumentationURL: "https://docs.example.com/v2",
				UpgradeDeadline:           "2025-06-01",
			},
			"3.0.0": {
				Message:                   "Changed stream format",
				MigrationDocumentationURL: "https://docs.example.com/v3",
			},
		},
	}

	t.Run("multi-version jump aggregates intermediate changes", func(t *testing.T) {
		result := DetectBreakingChanges("1.0.0", "3.0.0", metadata)
		assert.Len(t, result, 2)
		assert.Equal(t, "2.0.0", result[0].Version)
		assert.Equal(t, "Removed legacy auth", result[0].Message)
		assert.Equal(t, "https://docs.example.com/v2", result[0].MigrationDocumentationURL)
		assert.Equal(t, "2025-06-01", result[0].UpgradeDeadline)
		assert.Equal(t, "3.0.0", result[1].Version)
		assert.Equal(t, "Changed stream format", result[1].Message)
	})

	t.Run("current version is exclusive, target is inclusive", func(t *testing.T) {
		result := DetectBreakingChanges("2.0.0", "3.0.0", metadata)
		assert.Len(t, result, 1)
		assert.Equal(t, "3.0.0", result[0].Version)
	})

	t.Run("non-semver current tag returns nil", func(t *testing.T) {
		result := DetectBreakingChanges("latest", "3.0.0", metadata)
		assert.Nil(t, result)
	})

	t.Run("non-semver target tag returns nil", func(t *testing.T) {
		result := DetectBreakingChanges("1.0.0", "dev", metadata)
		assert.Nil(t, result)
	})

	t.Run("nil metadata returns nil", func(t *testing.T) {
		result := DetectBreakingChanges("1.0.0", "2.0.0", nil)
		assert.Nil(t, result)
	})

	t.Run("empty breaking changes map returns nil", func(t *testing.T) {
		result := DetectBreakingChanges("1.0.0", "2.0.0", &pipelineservice.RepositoryConnectorMetadata{
			BreakingChanges: map[string]pipelineservice.BreakingChange{},
		})
		assert.Nil(t, result)
	})

	t.Run("same version returns nil", func(t *testing.T) {
		result := DetectBreakingChanges("1.0.0", "1.0.0", metadata)
		assert.Nil(t, result)
	})
}
