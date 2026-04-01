package pipelineconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"gopkg.in/yaml.v3"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

const redactedPlaceholder = "***REDACTED***"

// WorkspaceConfig represents the YAML export schema.
type WorkspaceConfig struct {
	Version      string              `yaml:"version"`
	Sources      []SourceConfig      `yaml:"sources,omitempty"`
	Destinations []DestinationConfig `yaml:"destinations,omitempty"`
	Connections  []ConnectionConfig  `yaml:"connections,omitempty"`
}

// SourceConfig is a source entry in the export YAML.
type SourceConfig struct {
	Name               string         `yaml:"name"`
	ManagedConnectorID string         `yaml:"managed_connector_id"`
	ConnectorImage     string         `yaml:"connector_image,omitempty"`
	ConnectorTag       string         `yaml:"connector_tag,omitempty"`
	Config             map[string]any `yaml:"config,omitempty"`
}

// DestinationConfig is a destination entry in the export YAML.
type DestinationConfig struct {
	Name               string         `yaml:"name"`
	ManagedConnectorID string         `yaml:"managed_connector_id"`
	ConnectorImage     string         `yaml:"connector_image,omitempty"`
	ConnectorTag       string         `yaml:"connector_tag,omitempty"`
	Config             map[string]any `yaml:"config,omitempty"`
}

// ConnectionConfig is a connection entry in the export YAML.
type ConnectionConfig struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Schedule    string `yaml:"schedule,omitempty"`
	MaxAttempts int    `yaml:"max_attempts,omitempty"`
	Enabled     bool   `yaml:"enabled"`
}

// ExportConfigParams holds parameters for exporting workspace config.
type ExportConfigParams struct {
	WorkspaceID uuid.UUID
}

// ExportConfig exports the workspace configuration as YAML.
type ExportConfig struct {
	storage pipelineservice.Storage
}

// NewExportConfig creates a new ExportConfig use case.
func NewExportConfig(storage pipelineservice.Storage) *ExportConfig {
	return &ExportConfig{storage: storage}
}

// Execute exports all sources, destinations, and connections as YAML.
func (uc *ExportConfig) Execute(ctx context.Context, params ExportConfigParams) ([]byte, error) {
	sources, err := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	destinations, err := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	connections, err := uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	// Build a cache of managed connectors for image resolution.
	mcCache := make(map[uuid.UUID]*pipelineservice.ManagedConnector)

	sourceNameByID := make(map[uuid.UUID]string)
	destNameByID := make(map[uuid.UUID]string)

	cfg := WorkspaceConfig{Version: "1"}

	for _, source := range sources {
		sourceNameByID[source.ID] = source.Name
		connector := uc.resolveConnector(ctx, source.ManagedConnectorID, mcCache)

		sourceConfig := SourceConfig{
			Name:               source.Name,
			ManagedConnectorID: source.ManagedConnectorID.String(),
			Config:             redactSecrets(configToMap(source.Config)),
		}
		if connector != nil {
			sourceConfig.ConnectorImage = connector.DockerImage
			sourceConfig.ConnectorTag = connector.DockerTag
		}

		cfg.Sources = append(cfg.Sources, sourceConfig)
	}

	for _, dest := range destinations {
		destNameByID[dest.ID] = dest.Name
		connector := uc.resolveConnector(ctx, dest.ManagedConnectorID, mcCache)

		destConfig := DestinationConfig{
			Name:               dest.Name,
			ManagedConnectorID: dest.ManagedConnectorID.String(),
			Config:             redactSecrets(configToMap(dest.Config)),
		}
		if connector != nil {
			destConfig.ConnectorImage = connector.DockerImage
			destConfig.ConnectorTag = connector.DockerTag
		}

		cfg.Destinations = append(cfg.Destinations, destConfig)
	}

	for _, conn := range connections {
		schedule := ""
		if conn.Schedule != nil {
			schedule = *conn.Schedule
		}

		connConfig := ConnectionConfig{
			Source:      sourceNameByID[conn.SourceID],
			Destination: destNameByID[conn.DestinationID],
			Schedule:    schedule,
			MaxAttempts: conn.MaxAttempts,
			Enabled:     conn.Status == pipelineservice.ConnectionStatusActive,
		}
		cfg.Connections = append(cfg.Connections, connConfig)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling yaml: %w", err)
	}

	header := fmt.Sprintf("# Synclet workspace config — exported %s\n", time.Now().UTC().Format("2006-01-02"))

	return append([]byte(header), data...), nil
}

// resolveConnector loads a managed connector by ID, caching results.
func (uc *ExportConfig) resolveConnector(ctx context.Context, id uuid.UUID, cache map[uuid.UUID]*pipelineservice.ManagedConnector) *pipelineservice.ManagedConnector {
	if cached, ok := cache[id]; ok {
		return cached
	}

	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(id),
	})
	if err != nil {
		cache[id] = nil

		return nil
	}

	cache[id] = connector

	return connector
}

func configToMap(raw string) map[string]any {
	if raw == "" {
		return nil
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}

	return m
}

func redactSecrets(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	result := make(map[string]any, len(m))
	for key, value := range m {
		switch val := value.(type) {
		case string:
			if isLikelySecret(key) {
				result[key] = redactedPlaceholder
			} else {
				result[key] = val
			}
		case map[string]any:
			result[key] = redactSecrets(val)
		default:
			result[key] = value
		}
	}

	return result
}

var secretKeywords = []string{"password", "secret", "token", "key", "credential", "api_key", "apikey", "private"}

func isLikelySecret(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, kw := range secretKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}
