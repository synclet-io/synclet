package pipelineconnect

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

func TestConfigToProto_MasksSecrets(t *testing.T) {
	ref := secretutil.MakeSecretRef(uuid.New())
	config := `{"host":"db","credentials":{"password":"` + ref + `"}}`

	source := sourceToProto(&pipelineservice.Source{ID: uuid.New(), Config: config})
	destination := destinationToProto(&pipelineservice.Destination{ID: uuid.New(), Config: config})

	for name, cfg := range map[string]map[string]any{
		"source":      source.GetConfig().AsMap(),
		"destination": destination.GetConfig().AsMap(),
	} {
		credentials, ok := cfg["credentials"].(map[string]any)
		require.True(t, ok, name)
		assert.Equal(t, secretutil.SecretMask, credentials["password"], name)
		assert.Equal(t, "db", cfg["host"], name)
	}
}
