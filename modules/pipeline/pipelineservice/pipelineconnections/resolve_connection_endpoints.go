package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ConnectionEndpoints holds display names of a connection's source and
// destination and of their connectors. A name is empty when the referenced
// record is not found in the workspace.
type ConnectionEndpoints struct {
	SourceName               string
	SourceConnectorName      string
	DestinationName          string
	DestinationConnectorName string
}

// ResolveConnectionEndpointsParams holds parameters for resolving endpoint names.
type ResolveConnectionEndpointsParams struct {
	WorkspaceID uuid.UUID
	Connections []*pipelineservice.Connection
}

// ResolveConnectionEndpoints looks up source, destination and connector names
// for a batch of connections with one query per entity type.
type ResolveConnectionEndpoints struct {
	storage pipelineservice.Storage
}

// NewResolveConnectionEndpoints creates a new ResolveConnectionEndpoints use case.
func NewResolveConnectionEndpoints(storage pipelineservice.Storage) *ResolveConnectionEndpoints {
	return &ResolveConnectionEndpoints{storage: storage}
}

// Execute returns endpoint names keyed by connection ID.
func (uc *ResolveConnectionEndpoints) Execute(ctx context.Context, params ResolveConnectionEndpointsParams) (map[uuid.UUID]ConnectionEndpoints, error) {
	result := make(map[uuid.UUID]ConnectionEndpoints, len(params.Connections))
	if len(params.Connections) == 0 {
		return result, nil
	}

	sourceIDs := lo.Uniq(lo.Map(params.Connections, func(c *pipelineservice.Connection, _ int) uuid.UUID { return c.SourceID }))
	destIDs := lo.Uniq(lo.Map(params.Connections, func(c *pipelineservice.Connection, _ int) uuid.UUID { return c.DestinationID }))

	sources, err := uc.storage.Sources().Find(ctx, &pipelineservice.SourceFilter{
		ID:          filter.In(sourceIDs...),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	destinations, err := uc.storage.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.In(destIDs...),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	connectorIDs := lo.Uniq(append(
		lo.Map(sources, func(s *pipelineservice.Source, _ int) uuid.UUID { return s.ManagedConnectorID }),
		lo.Map(destinations, func(d *pipelineservice.Destination, _ int) uuid.UUID { return d.ManagedConnectorID })...,
	))

	connectorNames := map[uuid.UUID]string{}

	if len(connectorIDs) > 0 {
		connectors, err := uc.storage.ManagedConnectors().Find(ctx, &pipelineservice.ManagedConnectorFilter{
			ID: filter.In(connectorIDs...),
		})
		if err != nil {
			return nil, fmt.Errorf("listing managed connectors: %w", err)
		}

		for _, mc := range connectors {
			connectorNames[mc.ID] = mc.Name
		}
	}

	sourcesByID := lo.KeyBy(sources, func(s *pipelineservice.Source) uuid.UUID { return s.ID })
	destsByID := lo.KeyBy(destinations, func(d *pipelineservice.Destination) uuid.UUID { return d.ID })

	for _, conn := range params.Connections {
		var endpoints ConnectionEndpoints
		if src, ok := sourcesByID[conn.SourceID]; ok {
			endpoints.SourceName = src.Name
			endpoints.SourceConnectorName = connectorNames[src.ManagedConnectorID]
		}

		if dst, ok := destsByID[conn.DestinationID]; ok {
			endpoints.DestinationName = dst.Name
			endpoints.DestinationConnectorName = connectorNames[dst.ManagedConnectorID]
		}

		result[conn.ID] = endpoints
	}

	return result, nil
}
