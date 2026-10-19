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
