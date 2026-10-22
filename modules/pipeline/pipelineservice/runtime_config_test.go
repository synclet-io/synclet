package pipelineservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRuntimeConfig_ServiceAccountName(t *testing.T) {
	for _, name := range []string{"", "wi-sa", "team.sync-reader", "a"} {
		require.NoError(t, ValidateRuntimeConfig(&RuntimeConfig{ServiceAccountName: name}), name)
	}

	for _, name := range []string{"WI-SA", "wi_sa", "-wi", "wi-", "wi sa", "ns/wi"} {
		err := ValidateRuntimeConfig(&RuntimeConfig{ServiceAccountName: name})
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "service_account_name", name)
	}
}

func TestDecodeRuntimeConfig(t *testing.T) {
	t.Run("accepts a valid object", func(t *testing.T) {
		cfg, err := DecodeRuntimeConfig(`{"cpu_limit":"500m","memory_limit":"256Mi"}`)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "500m", cfg.CPULimit)
		assert.Equal(t, "256Mi", cfg.MemoryLimit)
	})

	t.Run("accepts an empty object", func(t *testing.T) {
		cfg, err := DecodeRuntimeConfig(`{}`)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	refusals := []struct {
		name  string
		raw   string
		wants []string
	}{
		{name: "invalid JSON", raw: `not json`, wants: []string{"not a JSON object"}},
		{name: "array", raw: `[1]`, wants: []string{"not a JSON object"}},
		{name: "string", raw: `"x"`, wants: []string{"not a JSON object"}},
		{name: "null", raw: `null`, wants: []string{"not a JSON object"}},
		{name: "trailing data", raw: `{"cpu_limit":"1"} {}`, wants: []string{"not a JSON object"}},
		{name: "unknown top-level key", raw: `{"cpu_limt":"1"}`, wants: []string{"cpu_limt"}},
		{
			name:  "unknown docker volume key",
			raw:   `{"docker_volumes":[{"host_path":"/a","container_path":"/b","mode":"ro"}]}`,
			wants: []string{"mode"},
		},
		{name: "invalid quantity", raw: `{"cpu_limit":"lots"}`, wants: []string{"invalid runtime config", "cpu_limit"}},
	}

	for _, tc := range refusals {
		t.Run("refuses "+tc.name, func(t *testing.T) {
			cfg, err := DecodeRuntimeConfig(tc.raw)
			require.Error(t, err)
			assert.Nil(t, cfg)

			for _, want := range tc.wants {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}
