package pipelineconfig

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"gopkg.in/yaml.v3"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/pkg/secretutil"
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

// ImportConfig imports YAML configuration into a workspace. Every entry is
// written through the same use case the API uses for it, so secrets are
// encrypted, cron expressions validated and per-entry audit events recorded.
type ImportConfig struct {
	storage           pipelineservice.Storage
	audit             pipelineservice.AuditRecorder
	createSource      *pipelinesources.CreateSource
	updateSource      *pipelinesources.UpdateSource
	createDestination *pipelinedestinations.CreateDestination
	updateDestination *pipelinedestinations.UpdateDestination
	createConnection  *pipelineconnections.CreateConnection
	updateConnection  *pipelineconnections.UpdateConnection
	enableConnection  *pipelineconnections.EnableConnection
	disableConnection *pipelineconnections.DisableConnection
}

// NewImportConfig creates a new ImportConfig use case.
func NewImportConfig(
	storage pipelineservice.Storage,
	audit pipelineservice.AuditRecorder,
	createSource *pipelinesources.CreateSource,
	updateSource *pipelinesources.UpdateSource,
	createDestination *pipelinedestinations.CreateDestination,
	updateDestination *pipelinedestinations.UpdateDestination,
	createConnection *pipelineconnections.CreateConnection,
	updateConnection *pipelineconnections.UpdateConnection,
	enableConnection *pipelineconnections.EnableConnection,
	disableConnection *pipelineconnections.DisableConnection,
) *ImportConfig {
	return &ImportConfig{
		storage:           storage,
		audit:             audit,
		createSource:      createSource,
		updateSource:      updateSource,
		createDestination: createDestination,
		updateDestination: updateDestination,
		createConnection:  createConnection,
		updateConnection:  updateConnection,
		enableConnection:  enableConnection,
		disableConnection: disableConnection,
	}
}

// Execute parses YAML config and creates/updates sources, destinations, and
// connections. Problems with a single entry are reported in the result and do
// not stop the import; storage failures abort it.
func (uc *ImportConfig) Execute(ctx context.Context, params ImportConfigParams) (*ImportConfigResult, error) {
	var cfg WorkspaceConfig
	if err := yaml.Unmarshal(params.ConfigYAML, &cfg); err != nil {
		return nil, &pipelineservice.ValidationError{Field: "config_yaml", Message: "invalid YAML: " + err.Error()}
	}

	if cfg.Version != "1" {
		return nil, &pipelineservice.ValidationError{
			Field:   "config_yaml",
			Message: fmt.Sprintf("unsupported config version %q, expected \"1\"", cfg.Version),
		}
	}

	imp := &importRun{uc: uc, workspaceID: params.WorkspaceID, result: &ImportConfigResult{}}

	for _, sourceConfig := range cfg.Sources {
		if err := imp.importSource(ctx, sourceConfig); err != nil {
			return nil, err
		}
	}

	for _, destConfig := range cfg.Destinations {
		if err := imp.importDestination(ctx, destConfig); err != nil {
			return nil, err
		}
	}

	for _, connConfig := range cfg.Connections {
		if err := imp.importConnection(ctx, connConfig); err != nil {
			return nil, err
		}
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "workspace",
		ResourceID:    params.WorkspaceID,
		ResourceLabel: "config import",
		After: map[string]any{
			"sources":      imp.sourcesWritten,
			"destinations": imp.destinationsWritten,
			"connections":  imp.connectionsWritten,
		},
	})

	return imp.result, nil
}

type importRun struct {
	uc          *ImportConfig
	workspaceID uuid.UUID
	result      *ImportConfigResult

	sourcesWritten      int
	destinationsWritten int
	connectionsWritten  int
}

func (r *importRun) fail(format string, args ...any) {
	r.result.Errors = append(r.result.Errors, fmt.Sprintf(format, args...))
}

func (r *importRun) importSource(ctx context.Context, entry SourceConfig) error {
	managedConnectorID, ok := r.parseConnectorEntry("source", entry.Name, entry.ManagedConnectorID)
	if !ok {
		return nil
	}

	existing, err := r.findSource(ctx, entry.Name)
	if err != nil {
		return err
	}

	if existing == nil {
		configJSON, err := createConfigJSON(entry.Config)
		if err != nil {
			r.fail("source %q: %v", entry.Name, err)

			return nil
		}

		if _, err := r.uc.createSource.Execute(ctx, pipelinesources.CreateSourceParams{
			WorkspaceID:        r.workspaceID,
			Name:               entry.Name,
			ManagedConnectorID: managedConnectorID,
			Config:             configJSON,
		}); err != nil {
			r.fail("creating source %q: %v", entry.Name, err)

			return nil
		}

		r.sourcesWritten++
		r.result.Created++

		return nil
	}

	if existing.ManagedConnectorID != managedConnectorID {
		r.fail("source %q: managed_connector_id cannot be changed by import", entry.Name)

		return nil
	}

	configJSON, err := updateConfigJSON(existing.Config, entry.Config)
	if err != nil {
		r.fail("source %q: %v", entry.Name, err)

		return nil
	}

	if _, err := r.uc.updateSource.Execute(ctx, pipelinesources.UpdateSourceParams{
		ID:          existing.ID,
		WorkspaceID: r.workspaceID,
		Config:      &configJSON,
	}); err != nil {
		r.fail("updating source %q: %v", entry.Name, err)

		return nil
	}

	r.sourcesWritten++
	r.result.Updated++

	return nil
}

func (r *importRun) importDestination(ctx context.Context, entry DestinationConfig) error {
	managedConnectorID, ok := r.parseConnectorEntry("destination", entry.Name, entry.ManagedConnectorID)
	if !ok {
		return nil
	}

	existing, err := r.findDestination(ctx, entry.Name)
	if err != nil {
		return err
	}

	if existing == nil {
		configJSON, err := createConfigJSON(entry.Config)
		if err != nil {
			r.fail("destination %q: %v", entry.Name, err)

			return nil
		}

		if _, err := r.uc.createDestination.Execute(ctx, pipelinedestinations.CreateDestinationParams{
			WorkspaceID:        r.workspaceID,
			Name:               entry.Name,
			ManagedConnectorID: managedConnectorID,
			Config:             configJSON,
		}); err != nil {
			r.fail("creating destination %q: %v", entry.Name, err)

			return nil
		}

		r.destinationsWritten++
		r.result.Created++

		return nil
	}

	if existing.ManagedConnectorID != managedConnectorID {
		r.fail("destination %q: managed_connector_id cannot be changed by import", entry.Name)

		return nil
	}

	configJSON, err := updateConfigJSON(existing.Config, entry.Config)
	if err != nil {
		r.fail("destination %q: %v", entry.Name, err)

		return nil
	}

	if _, err := r.uc.updateDestination.Execute(ctx, pipelinedestinations.UpdateDestinationParams{
		ID:          existing.ID,
		WorkspaceID: r.workspaceID,
		Config:      &configJSON,
	}); err != nil {
		r.fail("updating destination %q: %v", entry.Name, err)

		return nil
	}

	r.destinationsWritten++
	r.result.Updated++

	return nil
}

func (r *importRun) parseConnectorEntry(kind, name, managedConnectorID string) (uuid.UUID, bool) {
	if name == "" || managedConnectorID == "" {
		r.fail("%s missing required fields: name=%q managed_connector_id=%q", kind, name, managedConnectorID)

		return uuid.Nil, false
	}

	id, err := uuid.Parse(managedConnectorID)
	if err != nil {
		r.fail("%s %q: invalid managed_connector_id: %v", kind, name, err)

		return uuid.Nil, false
	}

	return id, true
}

func (r *importRun) importConnection(ctx context.Context, entry ConnectionConfig) error {
	if entry.Source == "" || entry.Destination == "" {
		r.fail("connection missing source or destination name")

		return nil
	}

	label := entry.Source + "->" + entry.Destination

	source, err := r.findSource(ctx, entry.Source)
	if err != nil {
		return err
	}

	if source == nil {
		r.fail("source %q not found for connection", entry.Source)

		return nil
	}

	destination, err := r.findDestination(ctx, entry.Destination)
	if err != nil {
		return err
	}

	if destination == nil {
		r.fail("destination %q not found for connection", entry.Destination)

		return nil
	}

	var schedule *string
	if entry.Schedule != "" {
		schedule = &entry.Schedule
	}

	status := pipelineservice.ConnectionStatusInactive
	if entry.Enabled {
		status = pipelineservice.ConnectionStatusActive
	}

	existing, err := r.uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		WorkspaceID:   filter.Equals(r.workspaceID),
		SourceID:      filter.Equals(source.ID),
		DestinationID: filter.Equals(destination.ID),
	})
	if err != nil {
		return fmt.Errorf("looking up connection %s: %w", label, err)
	}

	if len(existing) == 0 {
		if _, err := r.uc.createConnection.Execute(ctx, pipelineconnections.CreateConnectionParams{
			WorkspaceID:         r.workspaceID,
			Name:                fmt.Sprintf("%s -> %s", entry.Source, entry.Destination),
			SourceID:            source.ID,
			DestinationID:       destination.ID,
			Schedule:            schedule,
			SchemaChangePolicy:  pipelineservice.SchemaChangePolicyPropagate,
			MaxAttempts:         entry.MaxAttempts,
			NamespaceDefinition: pipelineservice.NamespaceDefinitionSource,
			Status:              status,
		}); err != nil {
			r.fail("creating connection %s: %v", label, err)

			return nil
		}

		r.connectionsWritten++
		r.result.Created++

		return nil
	}

	conn := existing[0]
	update := pipelineconnections.UpdateConnectionParams{
		ID:          conn.ID,
		WorkspaceID: r.workspaceID,
		Schedule:    &schedule,
	}

	if entry.MaxAttempts > 0 {
		update.MaxAttempts = &entry.MaxAttempts
	}

	if _, err := r.uc.updateConnection.Execute(ctx, update); err != nil {
		r.fail("updating connection %s: %v", label, err)

		return nil
	}

	if err := r.setConnectionStatus(ctx, conn, status); err != nil {
		r.fail("updating connection %s: %v", label, err)

		return nil
	}

	r.connectionsWritten++
	r.result.Updated++

	return nil
}

func (r *importRun) setConnectionStatus(ctx context.Context, conn *pipelineservice.Connection, status pipelineservice.ConnectionStatus) error {
	if conn.Status == status {
		return nil
	}

	if status == pipelineservice.ConnectionStatusActive {
		_, err := r.uc.enableConnection.Execute(ctx, pipelineconnections.EnableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  r.workspaceID,
		})

		return err
	}

	_, err := r.uc.disableConnection.Execute(ctx, pipelineconnections.DisableConnectionParams{
		ConnectionID: conn.ID,
		WorkspaceID:  r.workspaceID,
	})

	return err
}

func (r *importRun) findSource(ctx context.Context, name string) (*pipelineservice.Source, error) {
	found, err := r.uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
		WorkspaceID: filter.Equals(r.workspaceID),
		Name:        filter.Equals(name),
	})
	if err != nil {
		return nil, fmt.Errorf("looking up source %q: %w", name, err)
	}

	if len(found) == 0 {
		return nil, nil
	}

	return found[0], nil
}

func (r *importRun) findDestination(ctx context.Context, name string) (*pipelineservice.Destination, error) {
	found, err := r.uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID: filter.Equals(r.workspaceID),
		Name:        filter.Equals(name),
	})
	if err != nil {
		return nil, fmt.Errorf("looking up destination %q: %w", name, err)
	}

	if len(found) == 0 {
		return nil, nil
	}

	return found[0], nil
}

// createConfigJSON drops redacted placeholders, masks and secret refs from an
// imported config: a new entry has no stored secrets to resolve them against.
func createConfigJSON(config map[string]any) (json.RawMessage, error) {
	cleaned, _ := withoutSecretPlaceholders(config).(map[string]any)
	if cleaned == nil {
		cleaned = map[string]any{}
	}

	data, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}

	return data, nil
}

// updateConfigJSON overlays the imported top-level fields on the stored config
// in the masked form the update use cases accept: stored secret refs and
// redacted placeholders become secretutil.SecretMask, so they resolve to the
// stored secret, and fields missing from the import keep their stored value.
func updateConfigJSON(storedConfig string, incoming map[string]any) (json.RawMessage, error) {
	merged := map[string]any{}

	if storedConfig != "" {
		masked, err := pipelinesecrets.MaskConfigSecrets(storedConfig)
		if err != nil {
			return nil, fmt.Errorf("reading stored config: %w", err)
		}

		if err := json.Unmarshal([]byte(masked), &merged); err != nil {
			return nil, fmt.Errorf("reading stored config: %w", err)
		}
	}

	for key, value := range incoming {
		merged[key] = redactedToMask(value)
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}

	return data, nil
}

func withoutSecretPlaceholders(value any) any {
	switch val := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))

		for key, item := range val {
			if isSecretPlaceholder(item) {
				continue
			}

			out[key] = withoutSecretPlaceholders(item)
		}

		return out
	case []any:
		out := make([]any, 0, len(val))

		for _, item := range val {
			if isSecretPlaceholder(item) {
				continue
			}

			out = append(out, withoutSecretPlaceholders(item))
		}

		return out
	default:
		return value
	}
}

func isSecretPlaceholder(value any) bool {
	s, ok := value.(string)

	return ok && (s == redactedPlaceholder || s == secretutil.SecretMask || secretutil.IsSecretRef(s))
}

func redactedToMask(value any) any {
	switch val := value.(type) {
	case string:
		if val == redactedPlaceholder {
			return secretutil.SecretMask
		}

		return val
	case map[string]any:
		out := make(map[string]any, len(val))
		for key, item := range val {
			out[key] = redactedToMask(item)
		}

		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = redactedToMask(item)
		}

		return out
	default:
		return value
	}
}
