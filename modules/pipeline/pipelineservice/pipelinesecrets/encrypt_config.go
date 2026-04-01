package pipelinesecrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

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

// UpdateConfigSecrets handles config update with placeholder preservation.
// existingConfigJSON is the current stored config (with $secret:uuid refs).
// newConfigJSON is the incoming config (may have ******** placeholders or new values).
func UpdateConfigSecrets(ctx context.Context, secrets pipelineservice.SecretsProvider, ownerType string, ownerID uuid.UUID, existingConfigJSON, newConfigJSON, specJSON string) (string, error) {
	secretPaths, err := ExtractSecretPaths(specJSON)
	if err != nil {
		return "", fmt.Errorf("extracting secret paths: %w", err)
	}

	if len(secretPaths) == 0 {
		return newConfigJSON, nil
	}

	var existingMap map[string]any
	if err := json.Unmarshal([]byte(existingConfigJSON), &existingMap); err != nil {
		return "", fmt.Errorf("parsing existing config JSON: %w", err)
	}

	var newMap map[string]any
	if err := json.Unmarshal([]byte(newConfigJSON), &newMap); err != nil {
		return "", fmt.Errorf("parsing new config JSON: %w", err)
	}

	for _, path := range secretPaths {
		newVal, ok := GetNestedField(newMap, path)
		if !ok {
			continue
		}

		strVal, ok := newVal.(string)
		if !ok {
			continue
		}

		// Mask placeholder: preserve existing secret ref.
		if strVal == secretutil.SecretMask {
			existingVal, exists := GetNestedField(existingMap, path)
			if exists {
				SetNestedField(newMap, path, existingVal)
			}

			continue
		}

		// Already a secret ref: keep as-is.
		if secretutil.IsSecretRef(strVal) {
			continue
		}

		// New plaintext value: store new secret first, then delete old ref.
		// Store-before-delete ensures the old secret remains intact if StoreSecret fails.
		secretRef, err := secrets.StoreSecret(ctx, ownerType, ownerID, strVal)
		if err != nil {
			return "", fmt.Errorf("storing secret for field %q: %w", path, err)
		}

		SetNestedField(newMap, path, secretRef)

		// Clean up the old secret ref now that the new one is safely stored.
		if existingVal, exists := GetNestedField(existingMap, path); exists {
			if existingStr, ok := existingVal.(string); ok && secretutil.IsSecretRef(existingStr) {
				if delErr := secrets.DeleteSecret(ctx, existingStr); delErr != nil {
					// Log but don't fail the update for cleanup errors.
					zap.L().Warn("failed to delete old secret", zap.String("field", path), zap.Error(delErr))
				}
			}
		}
	}

	result, err := json.Marshal(newMap)
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
