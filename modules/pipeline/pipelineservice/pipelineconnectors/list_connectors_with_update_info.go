package pipelineconnectors

import (
	"context"
	"fmt"

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
	WorkspaceID uuid.UUID
}

// Execute returns all managed connectors for a workspace, each enriched with update info.
func (uc *ListConnectorsWithUpdateInfo) Execute(ctx context.Context, params ListConnectorsWithUpdateInfoParams) ([]ManagedConnectorWithUpdateInfo, error) {
	connectors, err := uc.storage.ManagedConnectors().Find(ctx, &pipelineservice.ManagedConnectorFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing managed connectors: %w", err)
	}

	// Collect unique repository IDs from connectors that have a repo link.
	repoIDSet := make(map[uuid.UUID]struct{})
	for _, mc := range connectors {
		if mc.RepositoryID != nil {
			repoIDSet[*mc.RepositoryID] = struct{}{}
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

		for _, rc := range repoConns {
			repoConnMap[repoKey{RepositoryID: rc.RepositoryID, DockerRepository: rc.DockerRepository}] = rc
		}
	}

	// Enrich each connector with update info.
	result := make([]ManagedConnectorWithUpdateInfo, len(connectors))
	for i, mc := range connectors {
		result[i] = ManagedConnectorWithUpdateInfo{Connector: mc}

		if mc.RepositoryID == nil {
			continue
		}

		rc, ok := repoConnMap[repoKey{RepositoryID: *mc.RepositoryID, DockerRepository: mc.DockerImage}]
		if !ok {
			continue
		}

		result[i].UpdateInfo = enrichUpdateInfo(mc, rc)
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
	mc, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting managed connector: %w", err)
	}

	result := &ManagedConnectorWithUpdateInfo{Connector: mc}

	if mc.RepositoryID == nil {
		return result, nil
	}

	rc, err := uc.storage.RepositoryConnectors().First(ctx, &pipelineservice.RepositoryConnectorFilter{
		RepositoryID:     filter.Equals(*mc.RepositoryID),
		DockerRepository: filter.Equals(mc.DockerImage),
	})
	if err != nil {
		return result, nil //nolint:nilerr // not-found is expected, return without update info
	}

	result.UpdateInfo = enrichUpdateInfo(mc, rc)

	return result, nil
}

// enrichUpdateInfo computes the ManagedConnectorUpdateInfo for a managed connector
// given its matching repository connector.
func enrichUpdateInfo(mc *pipelineservice.ManagedConnector, rc *pipelineservice.RepositoryConnector) *ManagedConnectorUpdateInfo {
	hasUpdate := mc.DockerTag != rc.DockerImageTag
	info := &ManagedConnectorUpdateInfo{
		AvailableVersion: rc.DockerImageTag,
		HasUpdate:        hasUpdate,
	}

	if hasUpdate {
		metadata, err := pipelineservice.UnmarshalMetadata(rc.Metadata)
		if err == nil {
			info.BreakingChanges = DetectBreakingChanges(mc.DockerTag, rc.DockerImageTag, metadata)
		}
	}

	return info
}
