package pipelineconnectors

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ManagedConnectorUpdateInfo holds update availability and breaking change data.
type ManagedConnectorUpdateInfo struct {
	AvailableVersion string
	HasUpdate        bool
	BreakingChanges  []VersionedBreakingChange
}

// ManagedConnectorWithUpdateInfo pairs a managed connector with its update info.
type ManagedConnectorWithUpdateInfo struct {
	Connector  *pipelineservice.ManagedConnector
	UpdateInfo *ManagedConnectorUpdateInfo
	// Used is true when a source or destination of the connector's workspace is built on it.
	Used bool
}

// ListConnectorsWithUpdateInfo returns managed connectors enriched with update info.
type ListConnectorsWithUpdateInfo struct {
	storage pipelineservice.Storage
}

// NewListConnectorsWithUpdateInfo creates a new ListConnectorsWithUpdateInfo use case.
func NewListConnectorsWithUpdateInfo(storage pipelineservice.Storage) *ListConnectorsWithUpdateInfo {
	return &ListConnectorsWithUpdateInfo{storage: storage}
}

// ListConnectorsWithUpdateInfoParams holds parameters for listing connectors with update info.
type ListConnectorsWithUpdateInfoParams struct {
	WorkspaceID        uuid.UUID
	FilterRepositoryID *string // nil = no filter, pointer to "" = NULL repo, pointer to UUID = specific repo
	Search             string  // "" = no filter
}

// Execute returns all managed connectors for a workspace, each enriched with update info.
func (uc *ListConnectorsWithUpdateInfo) Execute(ctx context.Context, params ListConnectorsWithUpdateInfoParams) ([]ManagedConnectorWithUpdateInfo, error) {
	connectorFilter := &pipelineservice.ManagedConnectorFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}

	if params.FilterRepositoryID != nil {
		if *params.FilterRepositoryID == "" {
			connectorFilter.RepositoryID = filter.Equals((*uuid.UUID)(nil))
		} else {
			repoID, err := uuid.Parse(*params.FilterRepositoryID)
			if err != nil {
				return nil, fmt.Errorf("invalid repository_id: %w", err)
			}

			connectorFilter.RepositoryID = filter.Equals(&repoID)
		}
	}

	connectors, err := uc.storage.ManagedConnectors().Find(ctx, connectorFilter)
	if err != nil {
		return nil, fmt.Errorf("listing managed connectors: %w", err)
	}

	if params.Search != "" {
		search := strings.ToLower(params.Search)

		var filtered []*pipelineservice.ManagedConnector

		for _, c := range connectors {
			if strings.Contains(strings.ToLower(c.Name), search) ||
				strings.Contains(strings.ToLower(c.DockerImage), search) {
				filtered = append(filtered, c)
			}
		}

		connectors = filtered
	}

	// Collect unique repository IDs from connectors that have a repo link.
	repoIDSet := make(map[uuid.UUID]struct{})

	for _, conn := range connectors {
		if conn.RepositoryID != nil {
			repoIDSet[*conn.RepositoryID] = struct{}{}
		}
	}

	// Batch-load all repo connectors for the set of repository IDs.
	type repoKey struct {
		RepositoryID     uuid.UUID
		DockerRepository string
	}
	repoConnMap := make(map[repoKey]*pipelineservice.RepositoryConnector)

	if len(repoIDSet) > 0 {
		repoIDs := make([]uuid.UUID, 0, len(repoIDSet))
		for id := range repoIDSet {
			repoIDs = append(repoIDs, id)
		}

		repoConns, findErr := uc.storage.RepositoryConnectors().Find(ctx, &pipelineservice.RepositoryConnectorFilter{
			RepositoryID: filter.In(repoIDs...),
		})
		if findErr != nil {
			return nil, fmt.Errorf("loading repository connectors: %w", findErr)
		}

		for _, repoConn := range repoConns {
			repoConnMap[repoKey{RepositoryID: repoConn.RepositoryID, DockerRepository: repoConn.DockerRepository}] = repoConn
		}
	}

	used, err := usedConnectorIDs(ctx, uc.storage, params.WorkspaceID, connectors)
	if err != nil {
		return nil, err
	}

	// Enrich each connector with update info.
	result := make([]ManagedConnectorWithUpdateInfo, len(connectors))
	for i, connector := range connectors {
		_, isUsed := used[connector.ID]
		result[i] = ManagedConnectorWithUpdateInfo{Connector: connector, Used: isUsed}

		if connector.RepositoryID == nil {
			continue
		}

		rc, ok := repoConnMap[repoKey{RepositoryID: *connector.RepositoryID, DockerRepository: connector.DockerImage}]
		if !ok {
			continue
		}

		result[i].UpdateInfo = enrichUpdateInfo(connector, rc)
	}

	return result, nil
}

// GetConnectorWithUpdateInfo returns a single managed connector enriched with update info.
type GetConnectorWithUpdateInfo struct {
	storage pipelineservice.Storage
}

// NewGetConnectorWithUpdateInfo creates a new GetConnectorWithUpdateInfo use case.
func NewGetConnectorWithUpdateInfo(storage pipelineservice.Storage) *GetConnectorWithUpdateInfo {
	return &GetConnectorWithUpdateInfo{storage: storage}
}

// GetConnectorWithUpdateInfoParams holds parameters for getting a connector with update info.
type GetConnectorWithUpdateInfoParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute returns a single managed connector enriched with update info.
func (uc *GetConnectorWithUpdateInfo) Execute(ctx context.Context, params GetConnectorWithUpdateInfoParams) (*ManagedConnectorWithUpdateInfo, error) {
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting managed connector: %w", err)
	}

	used, err := usedConnectorIDs(ctx, uc.storage, params.WorkspaceID, []*pipelineservice.ManagedConnector{connector})
	if err != nil {
		return nil, err
	}

	_, isUsed := used[connector.ID]
	result := &ManagedConnectorWithUpdateInfo{Connector: connector, Used: isUsed}

	if connector.RepositoryID == nil {
		return result, nil
	}

	repoConn, err := uc.storage.RepositoryConnectors().First(ctx, &pipelineservice.RepositoryConnectorFilter{
		RepositoryID:     filter.Equals(*connector.RepositoryID),
		DockerRepository: filter.Equals(connector.DockerImage),
	})
	if err != nil {
		return result, nil //nolint:nilerr // not-found is expected, return without update info
	}

	result.UpdateInfo = enrichUpdateInfo(connector, repoConn)

	return result, nil
}

// enrichUpdateInfo computes the ManagedConnectorUpdateInfo for a managed connector
// given its matching repository connector.
func enrichUpdateInfo(connector *pipelineservice.ManagedConnector, repoConn *pipelineservice.RepositoryConnector) *ManagedConnectorUpdateInfo {
	hasUpdate := connector.DockerTag != repoConn.DockerImageTag
	info := &ManagedConnectorUpdateInfo{
		AvailableVersion: repoConn.DockerImageTag,
		HasUpdate:        hasUpdate,
	}

	if hasUpdate {
		info.BreakingChanges = DetectBreakingChanges(connector.DockerTag, repoConn.DockerImageTag, &repoConn.Metadata)
	}

	return info
}

// usedConnectorIDs returns the IDs among connectors that a source or destination
// of the workspace is built on.
func usedConnectorIDs(ctx context.Context, storage pipelineservice.Storage, workspaceID uuid.UUID, connectors []*pipelineservice.ManagedConnector) (map[uuid.UUID]struct{}, error) {
	used := make(map[uuid.UUID]struct{})
	if len(connectors) == 0 {
		return used, nil
	}

	ids := make([]uuid.UUID, len(connectors))
	for i, connector := range connectors {
		ids[i] = connector.ID
	}

	sources, err := storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
		WorkspaceID:        filter.Equals(workspaceID),
		ManagedConnectorID: filter.In(ids...),
	})
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	for _, source := range sources {
		used[source.ManagedConnectorID] = struct{}{}
	}

	destinations, err := storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID:        filter.Equals(workspaceID),
		ManagedConnectorID: filter.In(ids...),
	})
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	for _, destination := range destinations {
		used[destination.ManagedConnectorID] = struct{}{}
	}

	return used, nil
}
