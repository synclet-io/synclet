package pipelineconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"gopkg.in/yaml.v3"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ImportConfigParams holds parameters for importing workspace config.
type ImportConfigParams struct {
	WorkspaceID uuid.UUID
	ConfigYAML  []byte
}

// ImportConfigResult contains the result of an import operation.
type ImportConfigResult struct {
	Created int
	Updated int
	Errors  []string
}

// ImportConfig imports YAML configuration into a workspace.
type ImportConfig struct {
	storage pipelineservice.Storage
}

// NewImportConfig creates a new ImportConfig use case.
func NewImportConfig(storage pipelineservice.Storage) *ImportConfig {
	return &ImportConfig{storage: storage}
}

// Execute parses YAML config and creates/updates sources, destinations, and connections.
func (uc *ImportConfig) Execute(ctx context.Context, params ImportConfigParams) (*ImportConfigResult, error) {
	var cfg WorkspaceConfig
	if err := yaml.Unmarshal(params.ConfigYAML, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if cfg.Version != "1" {
		return nil, fmt.Errorf("unsupported config version: %q (expected \"1\")", cfg.Version)
	}

	result := &ImportConfigResult{}

	// Import sources
	sourceIDByName := make(map[string]uuid.UUID)

	for _, sourceConfig := range cfg.Sources {
		if sourceConfig.Name == "" || sourceConfig.ManagedConnectorID == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("source missing required fields: name=%q managed_connector_id=%q", sourceConfig.Name, sourceConfig.ManagedConnectorID))

			continue
		}

		managedConnectorID, err := uuid.Parse(sourceConfig.ManagedConnectorID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("source %q: invalid managed_connector_id: %v", sourceConfig.Name, err))

			continue
		}

		configJSON := filterRedacted(sourceConfig.Config)
		configBytes, _ := json.Marshal(configJSON)

		existing, _ := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
			WorkspaceID: filter.Equals(params.WorkspaceID),
			Name:        filter.Equals(sourceConfig.Name),
		})

		if len(existing) > 0 {
			src := existing[0]
			sourceIDByName[sourceConfig.Name] = src.ID
			mergedConfig := mergeConfigs(configToMap(src.Config), sourceConfig.Config)
			mergedBytes, _ := json.Marshal(mergedConfig)
			src.ManagedConnectorID = managedConnectorID

			src.Config = string(mergedBytes)
			if _, err := uc.storage.Sources().Update(ctx, src); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("updating source %q: %v", sourceConfig.Name, err))

				continue
			}

			result.Updated++
		} else {
			src := pipelineservice.Source{
				ID:                 uuid.New(),
				WorkspaceID:        params.WorkspaceID,
				Name:               sourceConfig.Name,
				ManagedConnectorID: managedConnectorID,
				Config:             string(configBytes),
			}
			if _, err := uc.storage.Sources().Create(ctx, &src); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("creating source %q: %v", sourceConfig.Name, err))

				continue
			}

			sourceIDByName[sourceConfig.Name] = src.ID
			result.Created++
		}
	}

	// Import destinations
	destIDByName := make(map[string]uuid.UUID)

	for _, destConfig := range cfg.Destinations {
		if destConfig.Name == "" || destConfig.ManagedConnectorID == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("destination missing required fields: name=%q managed_connector_id=%q", destConfig.Name, destConfig.ManagedConnectorID))

			continue
		}

		managedConnectorID, err := uuid.Parse(destConfig.ManagedConnectorID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("destination %q: invalid managed_connector_id: %v", destConfig.Name, err))

			continue
		}

		configJSON := filterRedacted(destConfig.Config)
		configBytes, _ := json.Marshal(configJSON)

		existing, _ := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
			WorkspaceID: filter.Equals(params.WorkspaceID),
			Name:        filter.Equals(destConfig.Name),
		})

		if len(existing) > 0 {
			dst := existing[0]
			destIDByName[destConfig.Name] = dst.ID
			mergedConfig := mergeConfigs(configToMap(dst.Config), destConfig.Config)
			mergedBytes, _ := json.Marshal(mergedConfig)
			dst.ManagedConnectorID = managedConnectorID

			dst.Config = string(mergedBytes)
			if _, err := uc.storage.Destinations().Update(ctx, dst); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("updating destination %q: %v", destConfig.Name, err))

				continue
			}

			result.Updated++
		} else {
			dst := pipelineservice.Destination{
				ID:                 uuid.New(),
				WorkspaceID:        params.WorkspaceID,
				Name:               destConfig.Name,
				ManagedConnectorID: managedConnectorID,
				Config:             string(configBytes),
			}
			if _, err := uc.storage.Destinations().Create(ctx, &dst); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("creating destination %q: %v", destConfig.Name, err))

				continue
			}

			destIDByName[destConfig.Name] = dst.ID
			result.Created++
		}
	}

	// Import connections
	for _, connConfig := range cfg.Connections {
		if connConfig.Source == "" || connConfig.Destination == "" {
			result.Errors = append(result.Errors, "connection missing source or destination name")

			continue
		}

		sourceID, ok := sourceIDByName[connConfig.Source]
		if !ok {
			existing, _ := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
				WorkspaceID: filter.Equals(params.WorkspaceID),
				Name:        filter.Equals(connConfig.Source),
			})
			if len(existing) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("source %q not found for connection", connConfig.Source))

				continue
			}

			sourceID = existing[0].ID
		}

		destID, ok := destIDByName[connConfig.Destination]
		if !ok {
			existing, _ := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
				WorkspaceID: filter.Equals(params.WorkspaceID),
				Name:        filter.Equals(connConfig.Destination),
			})
			if len(existing) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("destination %q not found for connection", connConfig.Destination))

				continue
			}

			destID = existing[0].ID
		}

		var schedule *string
		if connConfig.Schedule != "" {
			schedule = &connConfig.Schedule
		}

		status := pipelineservice.ConnectionStatusInactive
		if connConfig.Enabled {
			status = pipelineservice.ConnectionStatusActive
		}

		existingConns, _ := uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
			WorkspaceID:   filter.Equals(params.WorkspaceID),
			SourceID:      filter.Equals(sourceID),
			DestinationID: filter.Equals(destID),
		})

		if len(existingConns) > 0 {
			conn := existingConns[0]
			conn.Schedule = schedule

			conn.Status = status
			if connConfig.MaxAttempts > 0 {
				conn.MaxAttempts = connConfig.MaxAttempts
			}

			pipelineservice.RecomputeNextScheduledAt(conn, time.Now())

			if _, err := uc.storage.Connections().Update(ctx, conn); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("updating connection %s->%s: %v", connConfig.Source, connConfig.Destination, err))

				continue
			}

			result.Updated++
		} else {
			maxAttempts := connConfig.MaxAttempts
			if maxAttempts == 0 {
				maxAttempts = 3
			}

			conn := pipelineservice.Connection{
				ID:                  uuid.New(),
				WorkspaceID:         params.WorkspaceID,
				Name:                fmt.Sprintf("%s -> %s", connConfig.Source, connConfig.Destination),
				Status:              status,
				SourceID:            sourceID,
				DestinationID:       destID,
				Schedule:            schedule,
				SchemaChangePolicy:  pipelineservice.SchemaChangePolicyPropagate,
				MaxAttempts:         maxAttempts,
				NamespaceDefinition: pipelineservice.NamespaceDefinitionSource,
			}
			pipelineservice.RecomputeNextScheduledAt(&conn, time.Now())

			if _, err := uc.storage.Connections().Create(ctx, &conn); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("creating connection %s->%s: %v", connConfig.Source, connConfig.Destination, err))

				continue
			}

			result.Created++
		}
	}

	return result, nil
}

// filterRedacted removes redacted placeholder values from a config map.
func filterRedacted(config map[string]any) map[string]any {
	if config == nil {
		return map[string]any{}
	}

	result := make(map[string]any, len(config))
	for k, v := range config {
		if s, ok := v.(string); ok && s == redactedPlaceholder {
			continue
		}

		result[k] = v
	}

	return result
}

// mergeConfigs merges new config into existing, keeping existing values where new has redacted placeholders.
func mergeConfigs(existing, incoming map[string]any) map[string]any {
	if existing == nil {
		return filterRedacted(incoming)
	}

	result := make(map[string]any, len(existing))
	for k, v := range existing {
		result[k] = v
	}

	for k, v := range incoming {
		if s, ok := v.(string); ok && s == redactedPlaceholder {
			continue // Keep existing value
		}

		result[k] = v
	}

	return result
}
