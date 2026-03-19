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
	for _, sc := range cfg.Sources {
		if sc.Name == "" || sc.ManagedConnectorID == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("source missing required fields: name=%q managed_connector_id=%q", sc.Name, sc.ManagedConnectorID))
			continue
		}

		managedConnectorID, err := uuid.Parse(sc.ManagedConnectorID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("source %q: invalid managed_connector_id: %v", sc.Name, err))
			continue
		}

		configJSON := filterRedacted(sc.Config)
		configBytes, _ := json.Marshal(configJSON)

		existing, _ := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
			WorkspaceID: filter.Equals(params.WorkspaceID),
			Name:        filter.Equals(sc.Name),
		})

		if len(existing) > 0 {
			src := existing[0]
			sourceIDByName[sc.Name] = src.ID
			mergedConfig := mergeConfigs(configToMap(src.Config), sc.Config)
			mergedBytes, _ := json.Marshal(mergedConfig)
			src.ManagedConnectorID = managedConnectorID
			src.Config = string(mergedBytes)
			if _, err := uc.storage.Sources().Update(ctx, src); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("updating source %q: %v", sc.Name, err))
				continue
			}
			result.Updated++
		} else {
			src := pipelineservice.Source{
				ID:                 uuid.New(),
				WorkspaceID:        params.WorkspaceID,
				Name:               sc.Name,
				ManagedConnectorID: managedConnectorID,
				Config:             string(configBytes),
			}
			if _, err := uc.storage.Sources().Create(ctx, &src); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("creating source %q: %v", sc.Name, err))
				continue
			}
			sourceIDByName[sc.Name] = src.ID
			result.Created++
		}
	}

	// Import destinations
	destIDByName := make(map[string]uuid.UUID)
	for _, dc := range cfg.Destinations {
		if dc.Name == "" || dc.ManagedConnectorID == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("destination missing required fields: name=%q managed_connector_id=%q", dc.Name, dc.ManagedConnectorID))
			continue
		}

		managedConnectorID, err := uuid.Parse(dc.ManagedConnectorID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("destination %q: invalid managed_connector_id: %v", dc.Name, err))
			continue
		}

		configJSON := filterRedacted(dc.Config)
		configBytes, _ := json.Marshal(configJSON)

		existing, _ := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
			WorkspaceID: filter.Equals(params.WorkspaceID),
			Name:        filter.Equals(dc.Name),
		})

		if len(existing) > 0 {
			dst := existing[0]
			destIDByName[dc.Name] = dst.ID
			mergedConfig := mergeConfigs(configToMap(dst.Config), dc.Config)
			mergedBytes, _ := json.Marshal(mergedConfig)
			dst.ManagedConnectorID = managedConnectorID
			dst.Config = string(mergedBytes)
			if _, err := uc.storage.Destinations().Update(ctx, dst); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("updating destination %q: %v", dc.Name, err))
				continue
			}
			result.Updated++
		} else {
			dst := pipelineservice.Destination{
				ID:                 uuid.New(),
				WorkspaceID:        params.WorkspaceID,
				Name:               dc.Name,
				ManagedConnectorID: managedConnectorID,
				Config:             string(configBytes),
			}
			if _, err := uc.storage.Destinations().Create(ctx, &dst); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("creating destination %q: %v", dc.Name, err))
				continue
			}
			destIDByName[dc.Name] = dst.ID
			result.Created++
		}
	}

	// Import connections
	for _, cc := range cfg.Connections {
		if cc.Source == "" || cc.Destination == "" {
			result.Errors = append(result.Errors, "connection missing source or destination name")
			continue
		}

		sourceID, ok := sourceIDByName[cc.Source]
		if !ok {
			existing, _ := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
				WorkspaceID: filter.Equals(params.WorkspaceID),
				Name:        filter.Equals(cc.Source),
			})
			if len(existing) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("source %q not found for connection", cc.Source))
				continue
			}
			sourceID = existing[0].ID
		}

		destID, ok := destIDByName[cc.Destination]
		if !ok {
			existing, _ := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
				WorkspaceID: filter.Equals(params.WorkspaceID),
				Name:        filter.Equals(cc.Destination),
			})
			if len(existing) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("destination %q not found for connection", cc.Destination))
				continue
			}
			destID = existing[0].ID
		}

		var schedule *string
		if cc.Schedule != "" {
			schedule = &cc.Schedule
		}

		status := pipelineservice.ConnectionStatusInactive
		if cc.Enabled {
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
			if cc.MaxAttempts > 0 {
				conn.MaxAttempts = cc.MaxAttempts
			}
			pipelineservice.RecomputeNextScheduledAt(conn, time.Now())
			if _, err := uc.storage.Connections().Update(ctx, conn); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("updating connection %s->%s: %v", cc.Source, cc.Destination, err))
				continue
			}
			result.Updated++
		} else {
			maxAttempts := cc.MaxAttempts
			if maxAttempts == 0 {
				maxAttempts = 3
			}
			conn := pipelineservice.Connection{
				ID:                  uuid.New(),
				WorkspaceID:         params.WorkspaceID,
				Name:                fmt.Sprintf("%s -> %s", cc.Source, cc.Destination),
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
				result.Errors = append(result.Errors, fmt.Sprintf("creating connection %s->%s: %v", cc.Source, cc.Destination, err))
				continue
			}
			result.Created++
		}
	}

	return result, nil
}

// filterRedacted removes redacted placeholder values from a config map.
func filterRedacted(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
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
