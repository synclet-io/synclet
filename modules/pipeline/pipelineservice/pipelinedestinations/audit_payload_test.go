package pipelinedestinations

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/pkg/auditutil"
)

func TestAuditConfig_MasksSpecFlaggedPlaintext(t *testing.T) {
	spec := `{"connectionSpecification":{"properties":{
		"client_id":{"type":"string","airbyte_secret":true},
		"ca_certificate":{"type":"string","airbyte_secret":true},
		"host":{"type":"string"}
	}}}`
	config := `{"client_id":"plain-id","ca_certificate":"$secret:7d3c3a0e-8c0e-4a59-9c8e-4f8f2a1b2c3d","host":"db","password":"pw"}`

	got, ok := auditConfig(config, spec).(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, auditutil.RedactMask, got["client_id"])
	assert.Equal(t, "$secret:7d3c3a0e-8c0e-4a59-9c8e-4f8f2a1b2c3d", got["ca_certificate"])
	assert.Equal(t, "db", got["host"])
	assert.Equal(t, auditutil.RedactMask, got["password"])
}

func TestAuditConfig_UnparsableSpecFallsBackToKeyNames(t *testing.T) {
	got, ok := auditConfig(`{"client_id":"plain-id","api_token":"t"}`, "not json").(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "plain-id", got["client_id"])
	assert.Equal(t, auditutil.RedactMask, got["api_token"])
}
