package pipelineservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	t.Run("refuses service_account_name as an unknown key", func(t *testing.T) {
		cfg, err := DecodeRuntimeConfig(`{"service_account_name":"wi-sa"}`)
		assert.Nil(t, cfg)

		var vErr *ValidationError
		require.ErrorAs(t, err, &vErr)
		assert.Equal(t, "runtime_config", vErr.Field)
		assert.Contains(t, vErr.Message, `unknown field "service_account_name"`)
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
			name:  "docker_volumes",
			raw:   `{"docker_volumes":[{"host_path":"/var/run/docker.sock","container_path":"/s"}]}`,
			wants: []string{"docker_volumes"},
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
