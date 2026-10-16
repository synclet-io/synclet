package pipelinesecrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// EncryptConfigSecrets encrypts secret fields in configJSON using specJSON to identify fields.
// For each field marked airbyte_secret in spec, calls secrets.StoreSecret and replaces value with reference.
func EncryptConfigSecrets(ctx context.Context, secrets pipelineservice.SecretsProvider, ownerType string, ownerID uuid.UUID, configJSON, specJSON string) (string, error) {
	secretPaths, err := ExtractSecretPaths(specJSON)
	if err != nil {
		return "", fmt.Errorf("extracting secret paths: %w", err)
	}

	if len(secretPaths) == 0 {
		return configJSON, nil
	}

	var configMap map[string]any
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return "", fmt.Errorf("parsing config JSON: %w", err)
	}

	for _, path := range secretPaths {
		val, ok := GetNestedField(configMap, path)
		if !ok {
			continue
		}

		strVal, ok := val.(string)
		if !ok || strVal == "" {
			continue
		}

		// Skip values that are already secret refs or mask placeholders.
		if secretutil.IsSecretRef(strVal) || strVal == secretutil.SecretMask {
			continue
		}

		secretRef, err := secrets.StoreSecret(ctx, ownerType, ownerID, strVal)
		if err != nil {
			return "", fmt.Errorf("storing secret for field %q: %w", path, err)
		}

		SetNestedField(configMap, path, secretRef)
	}

	result, err := json.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	return string(result), nil
}

// DecryptConfigSecrets resolves all $secret:uuid references in configJSON to plaintext.
func DecryptConfigSecrets(ctx context.Context, secrets pipelineservice.SecretsProvider, configJSON string) (string, error) {
	var configMap map[string]any
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return "", fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := decryptMap(ctx, secrets, configMap); err != nil {
		return "", err
	}

	result, err := json.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	return string(result), nil
}

// decryptMap recursively walks a map and replaces $secret: references with plaintext.
func decryptMap(ctx context.Context, secrets pipelineservice.SecretsProvider, obj map[string]any) error {
	for key, value := range obj {
		switch val := value.(type) {
		case string:
			if secretutil.IsSecretRef(val) {
				plaintext, err := secrets.RetrieveSecret(ctx, val)
				if err != nil {
					return fmt.Errorf("retrieving secret for field %q: %w", key, err)
				}

				obj[key] = plaintext
			}
		case map[string]any:
			if err := decryptMap(ctx, secrets, val); err != nil {
				return err
			}
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					if err := decryptMap(ctx, secrets, m); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
