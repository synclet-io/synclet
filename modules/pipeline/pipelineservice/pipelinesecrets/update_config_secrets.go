package pipelinesecrets

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

// MergeMaskedSecrets resolves an edited config against the stored one.
// Every secretutil.SecretMask value takes the stored value at the same
// position, or is dropped when nothing is stored there. A secret ref is
// accepted only where it equals the stored value, so a caller cannot point a
// config at someone else's secret. Fields absent from incomingConfigJSON stay
// absent and new plaintext values pass through unchanged.
func MergeMaskedSecrets(storedConfigJSON, incomingConfigJSON string) (string, error) {
	var stored map[string]any
	if storedConfigJSON != "" {
		if err := json.Unmarshal([]byte(storedConfigJSON), &stored); err != nil {
			return "", fmt.Errorf("parsing stored config JSON: %w", err)
		}
	}

	var incoming map[string]any
	if err := json.Unmarshal([]byte(incomingConfigJSON), &incoming); err != nil {
		return "", fmt.Errorf("parsing incoming config JSON: %w", err)
	}

	if err := mergeMaskedMap(stored, incoming, ""); err != nil {
		return "", err
	}

	result, err := json.Marshal(incoming)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	return string(result), nil
}

func mergeMaskedMap(stored, incoming map[string]any, prefix string) error {
	for key, value := range incoming {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		resolved, keep, err := mergeMaskedValue(stored[key], value, path)
		if err != nil {
			return err
		}

		if !keep {
			delete(incoming, key)

			continue
		}

		incoming[key] = resolved
	}

	return nil
}

func mergeMaskedValue(stored, incoming any, path string) (resolved any, keep bool, err error) {
	switch val := incoming.(type) {
	case string:
		storedStr, storedIsStr := stored.(string)

		switch {
		case val == secretutil.SecretMask:
			return storedStr, storedIsStr, nil
		case secretutil.IsSecretRef(val) && (!storedIsStr || storedStr != val):
			return nil, false, &pipelineservice.ValidationError{Field: path, Message: "unknown secret reference"}
		}
	case map[string]any:
		storedMap, _ := stored.(map[string]any)
		if mapErr := mergeMaskedMap(storedMap, val, path); mapErr != nil {
			return nil, false, mapErr
		}
	case []any:
		storedList, _ := stored.([]any)
		storedItems := matchStoredItems(storedList, val)

		for i, item := range val {
			merged, _, mergeErr := mergeMaskedValue(storedItems[i], item, path+"["+strconv.Itoa(i)+"]")
			if mergeErr != nil {
				return nil, false, mergeErr
			}

			val[i] = merged
		}
	}

	return incoming, true, nil
}

// matchStoredItems pairs every incoming array element with the stored element
// it was edited from. An element first pairs with the first unpaired stored
// element that equals it in every value that is not a mask; a bare mask pairs
// this way with nothing, as its stored counterpart cannot be told. When the
// arrays have equal length, an element still unpaired then takes the stored
// element at its own index if that one is unpaired too, which keeps an edited
// row's secrets.
func matchStoredItems(stored, incoming []any) []any {
	matched := make([]any, len(incoming))
	paired := make([]bool, len(incoming))
	used := make([]bool, len(stored))

	for i, item := range incoming {
		if item == secretutil.SecretMask {
			continue
		}

		for j, candidate := range stored {
			if !used[j] && equalIgnoringMasks(candidate, item) {
				used[j] = true
				paired[i] = true
				matched[i] = candidate

				break
			}
		}
	}

	if len(stored) != len(incoming) {
		return matched
	}

	for i := range incoming {
		if !paired[i] && !used[i] {
			used[i] = true
			matched[i] = stored[i]
		}
	}

	return matched
}

// equalIgnoringMasks reports whether incoming equals stored, treating every
// mask in incoming as equal to whatever is stored at its position.
func equalIgnoringMasks(stored, incoming any) bool {
	switch val := incoming.(type) {
	case string:
		if val == secretutil.SecretMask {
			return true
		}

		storedStr, ok := stored.(string)

		return ok && storedStr == val
	case map[string]any:
		storedMap, ok := stored.(map[string]any)
		if !ok {
			return false
		}

		for key, item := range val {
			storedItem, present := storedMap[key]
			if !present || !equalIgnoringMasks(storedItem, item) {
				return false
			}
		}

		return true
	case []any:
		storedList, ok := stored.([]any)
		if !ok || len(storedList) != len(val) {
			return false
		}

		for i, item := range val {
			if !equalIgnoringMasks(storedList[i], item) {
				return false
			}
		}

		return true
	default:
		return reflect.DeepEqual(stored, incoming)
	}
}

// UpdateConfigSecrets encrypts an updated config.
// existingConfigJSON is the current stored config (with $secret:uuid refs).
// newConfigJSON is the incoming config (may have ******** placeholders or new values).
// Masked values keep their stored ref, new plaintext values are stored as new
// secrets, and refs replaced or removed by the update are deleted.
func UpdateConfigSecrets(ctx context.Context, secrets pipelineservice.SecretsProvider, ownerType string, ownerID uuid.UUID, existingConfigJSON, newConfigJSON, specJSON string) (string, error) {
	mergedConfigJSON, err := MergeMaskedSecrets(existingConfigJSON, newConfigJSON)
	if err != nil {
		return "", err
	}

	secretPaths, err := ExtractSecretPaths(specJSON)
	if err != nil {
		return "", fmt.Errorf("extracting secret paths: %w", err)
	}

	if len(secretPaths) == 0 {
		return mergedConfigJSON, nil
	}

	var existingMap map[string]any
	if err := json.Unmarshal([]byte(existingConfigJSON), &existingMap); err != nil {
		return "", fmt.Errorf("parsing existing config JSON: %w", err)
	}

	var newMap map[string]any
	if err := json.Unmarshal([]byte(mergedConfigJSON), &newMap); err != nil {
		return "", fmt.Errorf("parsing new config JSON: %w", err)
	}

	// Store-before-delete keeps the old secrets intact if StoreSecret fails.
	for _, path := range ExpandSecretPaths(newMap, secretPaths) {
		newVal, _ := GetNestedField(newMap, path)

		strVal, _ := newVal.(string)
		if strVal == "" || secretutil.IsSecretRef(strVal) {
			continue
		}

		secretRef, err := secrets.StoreSecret(ctx, ownerType, ownerID, strVal)
		if err != nil {
			return "", fmt.Errorf("storing secret for field %q: %w", path, err)
		}

		SetNestedField(newMap, path, secretRef)
	}

	keptRefs := map[string]bool{}

	walkAndCollect(newMap, func(value string) {
		if secretutil.IsSecretRef(value) {
			keptRefs[value] = true
		}
	})

	var staleRefs []string

	walkAndCollect(existingMap, func(value string) {
		if secretutil.IsSecretRef(value) && !keptRefs[value] {
			keptRefs[value] = true
			staleRefs = append(staleRefs, value)
		}
	})

	result, err := json.Marshal(newMap)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	for _, ref := range staleRefs {
		if delErr := secrets.DeleteSecret(ctx, ref); delErr != nil {
			zap.L().Warn("failed to delete old secret", zap.Error(delErr))
		}
	}

	return string(result), nil
}
