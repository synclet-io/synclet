package pipelinesources

import (
	"encoding/json"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/pkg/auditutil"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// auditConfig converts a stored connector config into the redacted form recorded in audit events.
// Values under conventionally named secret keys are masked, and so are plaintext values at the
// spec's airbyte_secret paths; $secret references there stay visible so a diff shows the secret changed.
func auditConfig(configJSON, specJSON string) any {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil
	}

	specPaths, err := pipelinesecrets.ExtractSecretPaths(specJSON)
	if err != nil {
		specPaths = nil
	}

	plaintextPaths := make([]string, 0, len(specPaths))

	for _, path := range specPaths {
		if val, ok := pipelinesecrets.GetNestedField(cfg, path); ok {
			if ref, isStr := val.(string); !isStr || !secretutil.IsSecretRef(ref) {
				plaintextPaths = append(plaintextPaths, path)
			}
		}
	}

	return auditutil.Redact(cfg, plaintextPaths...)
}

// auditRuntimeConfig converts a nullable stored runtime config into the form recorded in audit events.
func auditRuntimeConfig(runtimeConfigJSON *string) any {
	if runtimeConfigJSON == nil || *runtimeConfigJSON == "" {
		return nil
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(*runtimeConfigJSON), &cfg); err != nil {
		return nil
	}

	return auditutil.Redact(cfg)
}
