package pipelineconnect

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	registryv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinerepositories"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// getConnectorVersionsUC abstracts the GetConnectorVersions use case for testability.
type getConnectorVersionsUC interface {
	Execute(ctx context.Context, params pipelinerepositories.GetConnectorVersionsParams) (*pipelinerepositories.GetConnectorVersionsResult, error)
}

// getConnectorUC abstracts the GetConnector use case for testability.
type getConnectorUC interface {
	Execute(ctx context.Context, params pipelineconnectors.GetConnectorParams) (*pipelineservice.ManagedConnector, error)
}

// getConnectorSpecUC abstracts the GetConnectorSpec use case for testability.
type getConnectorSpecUC interface {
	Execute(ctx context.Context, params pipelineconnectors.GetConnectorSpecParams) (*pipelineconnectors.GetConnectorSpecResult, error)
}

// deleteConnectorUC abstracts the DeleteConnector use case for testability.
type deleteConnectorUC interface {
	Execute(ctx context.Context, params pipelineconnectors.DeleteConnectorParams) error
}

// deleteRepositoryUC abstracts the DeleteRepository use case for testability.
type deleteRepositoryUC interface {
	Execute(ctx context.Context, params pipelinerepositories.DeleteRepositoryParams) (*pipelinerepositories.DeleteRepositoryResult, error)
}

// syncRepositoryUC abstracts the SyncRepository use case for testability.
type syncRepositoryUC interface {
	Execute(ctx context.Context, params pipelinerepositories.SyncRepositoryParams) (*pipelineservice.Repository, error)
}

// listRepoConnectorsUC abstracts the ListRepositoryConnectors use case for testability.
type listRepoConnectorsUC interface {
	Execute(ctx context.Context, params pipelinerepositories.ListRepositoryConnectorsParams) ([]*pipelineservice.RepositoryConnector, error)
}

// updateManagedConnectorUC abstracts the UpdateManagedConnector use case for testability.
type updateManagedConnectorUC interface {
	Execute(ctx context.Context, params pipelineconnectors.UpdateManagedConnectorParams) (*pipelineservice.ManagedConnector, error)
}

// batchUpdateConnectorsUC abstracts the BatchUpdateConnectors use case for testability.
type batchUpdateConnectorsUC interface {
	Execute(ctx context.Context, params pipelineconnectors.BatchUpdateConnectorsParams) (*pipelineconnectors.BatchUpdateConnectorsResult, error)
}

// listConnectorsWithUpdateInfoUC abstracts the ListConnectorsWithUpdateInfo use case.
type listConnectorsWithUpdateInfoUC interface {
	Execute(ctx context.Context, params pipelineconnectors.ListConnectorsWithUpdateInfoParams) ([]pipelineconnectors.ManagedConnectorWithUpdateInfo, error)
}

// getConnectorWithUpdateInfoUC abstracts the GetConnectorWithUpdateInfo use case.
type getConnectorWithUpdateInfoUC interface {
	Execute(ctx context.Context, params pipelineconnectors.GetConnectorWithUpdateInfoParams) (*pipelineconnectors.ManagedConnectorWithUpdateInfo, error)
}

// RegistryHandler implements the ConnectorRegistryService ConnectRPC handler.
type RegistryHandler struct {
	registryv1connect.UnimplementedConnectorRegistryServiceHandler
	addConnector                 *pipelineconnectors.AddConnector
	getConnector                 getConnectorUC
	getConnectorSpec             getConnectorSpecUC
	listConnectors               *pipelineconnectors.ListConnectors
	deleteConnector              deleteConnectorUC
	addRepository                *pipelinerepositories.AddRepository
	listRepositories             *pipelinerepositories.ListRepositories
	deleteRepository             deleteRepositoryUC
	syncRepository               syncRepositoryUC
	listRepoConnectors           listRepoConnectorsUC
	getConnectorVersions         getConnectorVersionsUC
	updateConnector              updateManagedConnectorUC
	batchUpdate                  batchUpdateConnectorsUC
	listConnectorsWithUpdateInfo listConnectorsWithUpdateInfoUC
	getConnectorWithUpdateInfo   getConnectorWithUpdateInfoUC
}

// NewRegistryHandler creates a new registry handler.
func NewRegistryHandler(
	addConnector *pipelineconnectors.AddConnector,
	getConnector *pipelineconnectors.GetConnector,
	getConnectorSpec *pipelineconnectors.GetConnectorSpec,
	listConnectors *pipelineconnectors.ListConnectors,
	deleteConnector *pipelineconnectors.DeleteConnector,
	addRepository *pipelinerepositories.AddRepository,
	listRepositories *pipelinerepositories.ListRepositories,
	deleteRepository *pipelinerepositories.DeleteRepository,
	syncRepository *pipelinerepositories.SyncRepository,
	listRepoConnectors *pipelinerepositories.ListRepositoryConnectors,
	getConnectorVersions *pipelinerepositories.GetConnectorVersions,
	updateConnector *pipelineconnectors.UpdateManagedConnector,
	batchUpdate *pipelineconnectors.BatchUpdateConnectors,
	listConnectorsWithUpdateInfo *pipelineconnectors.ListConnectorsWithUpdateInfo,
	getConnectorWithUpdateInfo *pipelineconnectors.GetConnectorWithUpdateInfo,
) *RegistryHandler {
	return &RegistryHandler{
		addConnector:                 addConnector,
		getConnector:                 getConnector,
		getConnectorSpec:             getConnectorSpec,
		listConnectors:               listConnectors,
		deleteConnector:              deleteConnector,
		addRepository:                addRepository,
		listRepositories:             listRepositories,
		deleteRepository:             deleteRepository,
		syncRepository:               syncRepository,
		listRepoConnectors:           listRepoConnectors,
		getConnectorVersions:         getConnectorVersions,
		updateConnector:              updateConnector,
		batchUpdate:                  batchUpdate,
		listConnectorsWithUpdateInfo: listConnectorsWithUpdateInfo,
		getConnectorWithUpdateInfo:   getConnectorWithUpdateInfo,
	}
}

// ListConnectors returns an empty list; use ListRepositoryConnectors instead.
func (h *RegistryHandler) ListConnectors(_ context.Context, _ *connect.Request[registryv1.ListConnectorsRequest]) (*connect.Response[registryv1.ListConnectorsResponse], error) {
	return connect.NewResponse(&registryv1.ListConnectorsResponse{
		Connectors: []*registryv1.ConnectorInfo{},
	}), nil
}

func (h *RegistryHandler) GetConnectorSpec(ctx context.Context, req *connect.Request[registryv1.GetConnectorSpecRequest]) (*connect.Response[registryv1.GetConnectorSpecResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connector id: %w", err))
	}

	result, err := h.getConnectorSpec.Execute(ctx, pipelineconnectors.GetConnectorSpecParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	spec, err := specJSONToProto(result.Spec)
	if err != nil {
		return nil, mapError(err)
	}

	var protoDocs []*registryv1.ExternalDocumentationUrl
	for _, d := range result.ExternalDocumentationURLs {
		protoDocs = append(protoDocs, &registryv1.ExternalDocumentationUrl{
			Title: d.Title,
			Type:  d.Type,
			Url:   d.URL,
		})
	}

	return connect.NewResponse(&registryv1.GetConnectorSpecResponse{
		Spec:                      spec,
		ExternalDocumentationUrls: protoDocs,
	}), nil
}

// AddConnector creates a managed connector and starts async pull+spec extraction.
func (h *RegistryHandler) AddConnector(ctx context.Context, req *connect.Request[registryv1.AddConnectorRequest]) (*connect.Response[registryv1.AddConnectorResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
		connectutil.StringValidation{Field: "docker_image", Value: req.Msg.GetDockerImage(), MaxLen: connectutil.MaxNameLength},
		connectutil.StringValidation{Field: "docker_tag", Value: req.Msg.GetDockerTag(), MaxLen: 128},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var repoID *uuid.UUID

	if req.Msg.GetRepositoryId() != "" {
		parsed, err := uuid.Parse(req.Msg.GetRepositoryId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid repository_id: %w", err))
		}

		repoID = &parsed
	}

	connector, err := h.addConnector.Execute(ctx, pipelineconnectors.AddConnectorParams{
		WorkspaceID:   workspaceID,
		DockerImage:   req.Msg.GetDockerImage(),
		DockerTag:     req.Msg.GetDockerTag(),
		Name:          req.Msg.GetName(),
		ConnectorType: protoToConnectorType(req.Msg.GetConnectorType()),
		RepositoryID:  repoID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&registryv1.AddConnectorResponse{
		Id: connector.ID.String(),
	}), nil
}

// GetManagedConnector returns a single managed connector by ID with update info.
func (h *RegistryHandler) GetManagedConnector(ctx context.Context, req *connect.Request[registryv1.GetManagedConnectorRequest]) (*connect.Response[registryv1.ManagedConnectorInfo], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connector id: %w", err))
	}

	item, err := h.getConnectorWithUpdateInfo.Execute(ctx, pipelineconnectors.GetConnectorWithUpdateInfoParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(managedConnectorWithUpdateInfoToProto(item)), nil
}

// ListManagedConnectors returns all managed connectors for the workspace with update info.
func (h *RegistryHandler) ListManagedConnectors(ctx context.Context, req *connect.Request[registryv1.ListManagedConnectorsRequest]) (*connect.Response[registryv1.ListManagedConnectorsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := pipelineconnectors.ListConnectorsWithUpdateInfoParams{
		WorkspaceID: workspaceID,
		Search:      req.Msg.Search,
	}
	if f := req.Msg.Filter; f != nil && f.RepositoryId != nil {
		params.FilterRepositoryID = f.RepositoryId
	}

	items, err := h.listConnectorsWithUpdateInfo.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	infos := make([]*registryv1.ManagedConnectorInfo, len(items))
	for i := range items {
		infos[i] = managedConnectorWithUpdateInfoToProto(&items[i])
	}

	return connect.NewResponse(&registryv1.ListManagedConnectorsResponse{
		Connectors: infos,
	}), nil
}

// DeleteManagedConnector removes a managed connector from the registry.
func (h *RegistryHandler) DeleteManagedConnector(ctx context.Context, req *connect.Request[registryv1.DeleteManagedConnectorRequest]) (*connect.Response[registryv1.DeleteManagedConnectorResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connector id: %w", err))
	}

	if err := h.deleteConnector.Execute(ctx, pipelineconnectors.DeleteConnectorParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&registryv1.DeleteManagedConnectorResponse{}), nil
}

// AddRepository creates a new connector repository (admin-only).
func (h *RegistryHandler) AddRepository(ctx context.Context, req *connect.Request[registryv1.AddRepositoryRequest]) (*connect.Response[registryv1.AddRepositoryResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
		connectutil.StringValidation{Field: "url", Value: req.Msg.GetUrl(), MaxLen: connectutil.MaxURLLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var authHeader *string
	if req.Msg.GetAuthHeader() != "" {
		authHeader = &req.Msg.AuthHeader
	}

	repo, err := h.addRepository.Execute(ctx, pipelinerepositories.AddRepositoryParams{
		WorkspaceID: workspaceID,
		Name:        req.Msg.GetName(),
		URL:         req.Msg.GetUrl(),
		AuthHeader:  authHeader,
	})
	if err != nil {
		return nil, mapError(fmt.Errorf("adding repository: %w", err))
	}

	return connect.NewResponse(&registryv1.AddRepositoryResponse{
		Repository: repositoryToProto(repo),
	}), nil
}

// ListRepositories returns all repositories for the workspace.
func (h *RegistryHandler) ListRepositories(ctx context.Context, _ *connect.Request[registryv1.ListRepositoriesRequest]) (*connect.Response[registryv1.ListRepositoriesResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	repos, err := h.listRepositories.Execute(ctx, pipelinerepositories.ListRepositoriesParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(fmt.Errorf("listing repositories: %w", err))
	}

	protoRepos := make([]*registryv1.Repository, len(repos))
	for i, r := range repos {
		protoRepos[i] = repositoryToProto(r)
	}

	return connect.NewResponse(&registryv1.ListRepositoriesResponse{
		Repositories: protoRepos,
	}), nil
}

// DeleteRepository removes a repository and disassociates managed connectors (admin-only).
func (h *RegistryHandler) DeleteRepository(ctx context.Context, req *connect.Request[registryv1.DeleteRepositoryRequest]) (*connect.Response[registryv1.DeleteRepositoryResponse], error) {
	repoID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid repository id: %w", err))
	}

	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	result, err := h.deleteRepository.Execute(ctx, pipelinerepositories.DeleteRepositoryParams{
		RepositoryID: repoID,
		WorkspaceID:  workspaceID,
	})
	if err != nil {
		return nil, mapError(fmt.Errorf("deleting repository: %w", err))
	}

	return connect.NewResponse(&registryv1.DeleteRepositoryResponse{
		AffectedConnectors: int32(result.AffectedConnectors),
	}), nil
}

// SyncRepository triggers a manual sync of a repository (admin-only).
func (h *RegistryHandler) SyncRepository(ctx context.Context, req *connect.Request[registryv1.SyncRepositoryRequest]) (*connect.Response[registryv1.SyncRepositoryResponse], error) {
	repoID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid repository id: %w", err))
	}

	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	repo, err := h.syncRepository.Execute(ctx, pipelinerepositories.SyncRepositoryParams{
		RepositoryID: repoID,
		WorkspaceID:  workspaceID,
	})
	if err != nil {
		return nil, mapError(fmt.Errorf("syncing repository: %w", err))
	}

	return connect.NewResponse(&registryv1.SyncRepositoryResponse{
		Repository: repositoryToProto(repo),
	}), nil
}

// ListRepositoryConnectors returns connectors from a specific repository.
func (h *RegistryHandler) ListRepositoryConnectors(ctx context.Context, req *connect.Request[registryv1.ListRepositoryConnectorsRequest]) (*connect.Response[registryv1.ListRepositoryConnectorsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	repoID, err := uuid.Parse(req.Msg.GetRepositoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid repository_id: %w", err))
	}

	params := pipelinerepositories.ListRepositoryConnectorsParams{
		RepositoryID: repoID,
		WorkspaceID:  workspaceID,
		Search:       req.Msg.GetSearch(),
	}
	if req.Msg.GetType() != registryv1.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED {
		params.ConnectorType = protoToConnectorType(req.Msg.GetType()).String()
	}

	if f := req.Msg.GetFilter(); f != nil {
		params.SupportLevel = protoToSupportLevel(f.GetSupportLevel())
		params.License = protoToLicense(f.GetLicense())
		params.SourceType = protoToSourceType(f.GetSourceType())
	}

	connectors, err := h.listRepoConnectors.Execute(ctx, params)
	if err != nil {
		return nil, mapError(fmt.Errorf("listing repository connectors: %w", err))
	}

	infos := make([]*registryv1.ConnectorInfo, len(connectors))
	for i, connector := range connectors {
		infos[i] = &registryv1.ConnectorInfo{
			DockerImage:   connector.DockerRepository,
			Name:          connector.Name,
			IconUrl:       connector.IconURL,
			DocsUrl:       connector.DocumentationURL,
			ReleaseStage:  releaseStageToProto(connector.ReleaseStage),
			LatestVersion: connector.DockerImageTag,
			Type:          managedConnectorTypeToProto(connector.ConnectorType),
			SupportLevel:  supportLevelToProto(connector.SupportLevel),
			License:       licenseStringToProto(connector.License),
			SourceType:    sourceTypeToProto(connector.SourceType),
		}
	}

	return connect.NewResponse(&registryv1.ListRepositoryConnectorsResponse{
		Connectors: infos,
	}), nil
}

// GetConnectorVersions returns available versions for a connector image.
func (h *RegistryHandler) GetConnectorVersions(ctx context.Context, req *connect.Request[registryv1.GetConnectorVersionsRequest]) (*connect.Response[registryv1.GetConnectorVersionsResponse], error) {
	if req.Msg.GetConnectorImage() == "" {
		return connect.NewResponse(&registryv1.GetConnectorVersionsResponse{}), nil
	}

	result, err := h.getConnectorVersions.Execute(ctx, pipelinerepositories.GetConnectorVersionsParams{
		ConnectorImage: req.Msg.GetConnectorImage(),
	})
	if err != nil {
		return nil, mapError(fmt.Errorf("getting connector versions: %w", err))
	}

	return connect.NewResponse(&registryv1.GetConnectorVersionsResponse{
		Versions:      result.Versions,
		LatestVersion: result.LatestVersion,
	}), nil
}

// UpdateManagedConnector updates a managed connector to the latest version from its repository.
func (h *RegistryHandler) UpdateManagedConnector(ctx context.Context, req *connect.Request[registryv1.UpdateManagedConnectorRequest]) (*connect.Response[registryv1.UpdateManagedConnectorResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connectorID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connector id: %w", err))
	}

	connector, err := h.updateConnector.Execute(ctx, pipelineconnectors.UpdateManagedConnectorParams{
		ConnectorID: connectorID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&registryv1.UpdateManagedConnectorResponse{
		Connector: managedConnectorToProto(connector),
	}), nil
}

// BatchUpdateConnectors updates explicitly requested connectors in the workspace.
func (h *RegistryHandler) BatchUpdateConnectors(ctx context.Context, req *connect.Request[registryv1.BatchUpdateConnectorsRequest]) (*connect.Response[registryv1.BatchUpdateConnectorsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	var connectorIDs []uuid.UUID

	for _, idStr := range req.Msg.GetConnectorIds() {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connector id %q: %w", idStr, err))
		}

		connectorIDs = append(connectorIDs, id)
	}

	result, err := h.batchUpdate.Execute(ctx, pipelineconnectors.BatchUpdateConnectorsParams{
		WorkspaceID:  workspaceID,
		ConnectorIDs: connectorIDs,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoConnectors := make([]*registryv1.ManagedConnectorInfo, len(result.UpdatedConnectors))
	for i, mc := range result.UpdatedConnectors {
		protoConnectors[i] = managedConnectorToProto(mc)
	}

	return connect.NewResponse(&registryv1.BatchUpdateConnectorsResponse{
		UpdatedCount:      int32(result.UpdatedCount),
		UpdatedConnectors: protoConnectors,
	}), nil
}
