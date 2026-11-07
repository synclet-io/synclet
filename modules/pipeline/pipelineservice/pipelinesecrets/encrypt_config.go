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

	for _, path := range ExpandSecretPaths(configMap, secretPaths) {
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

// DecryptConfigSecrets resolves all $secret:uuid references in configJSON to
// plaintext. Every reference must be held by ownerType/ownerID, the owner of
// the config.
func DecryptConfigSecrets(ctx context.Context, secrets pipelineservice.SecretsProvider, ownerType string, ownerID uuid.UUID, configJSON string) (string, error) {
	owner := secretOwner{ownerType: ownerType, ownerID: ownerID}

	var configMap map[string]any
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return "", fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := decryptMap(ctx, secrets, owner, configMap); err != nil {
		return "", err
	}

	result, err := json.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	return string(result), nil
}

// decryptMap recursively walks a map, including every array element, and
// replaces $secret: references with plaintext.
func decryptMap(ctx context.Context, secrets pipelineservice.SecretsProvider, owner secretOwner, obj map[string]any) error {
	for key, value := range obj {
		decrypted, err := decryptValue(ctx, secrets, owner, key, value)
		if err != nil {
			return err
		}

		obj[key] = decrypted
	}

	return nil
}

func decryptValue(ctx context.Context, secrets pipelineservice.SecretsProvider, owner secretOwner, key string, value any) (any, error) {
	switch val := value.(type) {
	case string:
		if secretutil.IsSecretRef(val) {
			plaintext, err := secrets.RetrieveSecret(ctx, owner.ownerType, owner.ownerID, val)
			if err != nil {
				return nil, fmt.Errorf("retrieving secret for field %q: %w", key, err)
			}

			return plaintext, nil
		}
	case map[string]any:
		if err := decryptMap(ctx, secrets, owner, val); err != nil {
			return nil, err
		}
	case []any:
		for i, item := range val {
			decrypted, err := decryptValue(ctx, secrets, owner, key, item)
			if err != nil {
				return nil, err
			}

			val[i] = decrypted
		}
	}

	return value, nil
}

// secretOwner names the owner every secret of a config must be held by.
type secretOwner struct {
	ownerType string
	ownerID   uuid.UUID
}
