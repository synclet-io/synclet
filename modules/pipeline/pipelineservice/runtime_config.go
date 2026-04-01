package pipelineservice

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// RuntimeConfig holds container runtime configuration for a source or destination.
// All fields are optional -- empty string means "use global default".
type RuntimeConfig struct {
	CPURequest         string          `json:"cpu_request,omitempty"`
	CPULimit           string          `json:"cpu_limit,omitempty"`
	MemoryRequest      string          `json:"memory_request,omitempty"`
	MemoryLimit        string          `json:"memory_limit,omitempty"`
	Tolerations        json.RawMessage `json:"tolerations,omitempty"`
	NodeSelector       json.RawMessage `json:"node_selector,omitempty"`
	Affinity           json.RawMessage `json:"affinity,omitempty"`
	ServiceAccountName string          `json:"service_account_name,omitempty"`
}

// RuntimeDefaults holds global default resource values loaded from env vars.
type RuntimeDefaults struct {
	CPURequest         string `env:"DEFAULT_CPU_REQUEST" envDefault:""`
	CPULimit           string `env:"DEFAULT_CPU_LIMIT" envDefault:""`
	MemoryRequest      string `env:"DEFAULT_MEMORY_REQUEST" envDefault:""`
	MemoryLimit        string `env:"DEFAULT_MEMORY_LIMIT" envDefault:""`
	ServiceAccountName string `env:"DEFAULT_SERVICE_ACCOUNT_NAME" envDefault:""`
}

// ParseRuntimeConfig parses a nullable JSONB string from DB model field.
// Returns nil if input is nil or empty.
func ParseRuntimeConfig(jsonb *string) *RuntimeConfig {
	if jsonb == nil || *jsonb == "" {
		return nil
	}

	var cfg RuntimeConfig
	if err := json.Unmarshal([]byte(*jsonb), &cfg); err != nil {
		return nil
	}

	return &cfg
}

// ResolveRuntimeConfig starts with defaults, then overlays non-empty override fields.
// K8s scheduling fields (tolerations, nodeSelector, affinity) replace entirely when present.
func ResolveRuntimeConfig(defaults RuntimeDefaults, override *RuntimeConfig) RuntimeConfig {
	result := RuntimeConfig{
		CPURequest:         defaults.CPURequest,
		CPULimit:           defaults.CPULimit,
		MemoryRequest:      defaults.MemoryRequest,
		MemoryLimit:        defaults.MemoryLimit,
		ServiceAccountName: defaults.ServiceAccountName,
	}

	if override == nil {
		return result
	}

	if override.CPURequest != "" {
		result.CPURequest = override.CPURequest
	}

	if override.CPULimit != "" {
		result.CPULimit = override.CPULimit
	}

	if override.MemoryRequest != "" {
		result.MemoryRequest = override.MemoryRequest
	}

	if override.MemoryLimit != "" {
		result.MemoryLimit = override.MemoryLimit
	}

	if len(override.Tolerations) > 0 {
		result.Tolerations = override.Tolerations
	}

	if len(override.NodeSelector) > 0 {
		result.NodeSelector = override.NodeSelector
	}

	if len(override.Affinity) > 0 {
		result.Affinity = override.Affinity
	}

	if override.ServiceAccountName != "" {
		result.ServiceAccountName = override.ServiceAccountName
	}

	return result
}

// ValidateRuntimeConfig validates all fields of a RuntimeConfig.
// Returns an aggregated error with all invalid field names.
func ValidateRuntimeConfig(cfg *RuntimeConfig) error {
	if cfg == nil {
		return nil
	}

	var errs []string

	if cfg.CPURequest != "" {
		if _, err := resource.ParseQuantity(cfg.CPURequest); err != nil {
			errs = append(errs, fmt.Sprintf("cpu_request: %v", err))
		}
	}

	if cfg.CPULimit != "" {
		if _, err := resource.ParseQuantity(cfg.CPULimit); err != nil {
			errs = append(errs, fmt.Sprintf("cpu_limit: %v", err))
		}
	}

	if cfg.MemoryRequest != "" {
		if _, err := resource.ParseQuantity(cfg.MemoryRequest); err != nil {
			errs = append(errs, fmt.Sprintf("memory_request: %v", err))
		}
	}

	if cfg.MemoryLimit != "" {
		if _, err := resource.ParseQuantity(cfg.MemoryLimit); err != nil {
			errs = append(errs, fmt.Sprintf("memory_limit: %v", err))
		}
	}

	if len(cfg.Tolerations) > 0 {
		var tolerations []corev1.Toleration
		if err := json.Unmarshal(cfg.Tolerations, &tolerations); err != nil {
			errs = append(errs, fmt.Sprintf("tolerations: must be a JSON array of Toleration objects: %v", err))
		}
	}

	if len(cfg.NodeSelector) > 0 {
		var ns map[string]string
		if err := json.Unmarshal(cfg.NodeSelector, &ns); err != nil {
			errs = append(errs, fmt.Sprintf("node_selector: must be a JSON object with string values: %v", err))
		}
	}

	if len(cfg.Affinity) > 0 {
		var affinity corev1.Affinity
		if err := json.Unmarshal(cfg.Affinity, &affinity); err != nil {
			errs = append(errs, fmt.Sprintf("affinity: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid runtime config: %s", strings.Join(errs, "; "))
	}

	return nil
}

// ToContainerResources converts string quantities to numeric values used by container.RunOptions
// and k8s.SyncOptions. Returns (memoryLimit, cpuLimit, memoryRequest, cpuRequest).
func ToContainerResources(cfg RuntimeConfig) (memoryLimit int64, cpuLimit float64, memoryRequest int64, cpuRequest float64) {
	if cfg.MemoryLimit != "" {
		if q, err := resource.ParseQuantity(cfg.MemoryLimit); err == nil {
			memoryLimit = q.Value()
		}
	}

	if cfg.CPULimit != "" {
		if q, err := resource.ParseQuantity(cfg.CPULimit); err == nil {
			cpuLimit = q.AsApproximateFloat64()
		}
	}

	if cfg.MemoryRequest != "" {
		if q, err := resource.ParseQuantity(cfg.MemoryRequest); err == nil {
			memoryRequest = q.Value()
		}
	}

	if cfg.CPURequest != "" {
		if q, err := resource.ParseQuantity(cfg.CPURequest); err == nil {
			cpuRequest = q.AsApproximateFloat64()
		}
	}

	return
}
