package pipelinedestinations

import (
	"encoding/json"

	"github.com/synclet-io/synclet/pkg/auditutil"
)

// auditConfig converts a stored connector config into the redacted form recorded in audit events.
// Only the stored config is ever recorded: spec-flagged secrets are already $secret references there, and Redact masks conventionally named secret keys on top.
func auditConfig(configJSON string) any {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil
	}

	return auditutil.Redact(cfg)
}

// auditRuntimeConfig converts a nullable stored runtime config into the form recorded in audit events.
func auditRuntimeConfig(runtimeConfigJSON *string) any {
	if runtimeConfigJSON == nil || *runtimeConfigJSON == "" {
		return nil
	}

	return auditConfig(*runtimeConfigJSON)
}
