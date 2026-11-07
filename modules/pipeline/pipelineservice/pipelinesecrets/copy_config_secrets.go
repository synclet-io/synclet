package pipelinesecrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// CopyConfigSecrets stores the plaintext behind every $secret: reference in
// configJSON, at any depth and whatever the connector spec marks, as a new
// secret of ownerType/ownerID and returns the config with the new references.
// The original secrets are left untouched. On error some secrets may already
// be stored under ownerID; the caller removes them with DeleteByOwner.
func CopyConfigSecrets(ctx context.Context, secrets pipelineservice.SecretsProvider, ownerType string, ownerID uuid.UUID, configJSON string) (string, error) {
	var config any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", fmt.Errorf("parsing config JSON: %w", err)
	}

	copied, err := copySecretRefs(ctx, secrets, ownerType, ownerID, config)
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(copied)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	return string(result), nil
}

func copySecretRefs(ctx context.Context, secrets pipelineservice.SecretsProvider, ownerType string, ownerID uuid.UUID, value any) (any, error) {
	switch val := value.(type) {
	case string:
		if !secretutil.IsSecretRef(val) {
			return val, nil
		}

		plaintext, err := secrets.RetrieveSecret(ctx, val)
		if err != nil {
			return nil, fmt.Errorf("retrieving secret: %w", err)
		}

		ref, err := secrets.StoreSecret(ctx, ownerType, ownerID, plaintext)
		if err != nil {
			return nil, fmt.Errorf("storing secret: %w", err)
		}

		return ref, nil
	case map[string]any:
		for key, item := range val {
			copied, err := copySecretRefs(ctx, secrets, ownerType, ownerID, item)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", key, err)
			}

			val[key] = copied
		}
	case []any:
		for i, item := range val {
			copied, err := copySecretRefs(ctx, secrets, ownerType, ownerID, item)
			if err != nil {
				return nil, err
			}

			val[i] = copied
		}
	}

	return value, nil
}
