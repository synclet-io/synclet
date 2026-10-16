package pipelinecatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// PendingSchemaChangesTarget identifies a connection and the source its
// configured catalog is compared against.
type PendingSchemaChangesTarget struct {
	ConnectionID uuid.UUID
	SourceID     uuid.UUID
}

// CountPendingSchemaChanges counts, for many connections at once, the schema
// changes DetectSchemaChanges would report for each of them.
type CountPendingSchemaChanges struct {
	storage pipelineservice.Storage
}

// NewCountPendingSchemaChanges creates a new CountPendingSchemaChanges use case.
func NewCountPendingSchemaChanges(storage pipelineservice.Storage) *CountPendingSchemaChanges {
	return &CountPendingSchemaChanges{storage: storage}
}

// Execute returns the pending change count per connection ID. Connections
// without a configured catalog or whose source was never discovered are
// absent from the result.
func (uc *CountPendingSchemaChanges) Execute(ctx context.Context, targets []PendingSchemaChangesTarget) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(targets))
	if len(targets) == 0 {
		return counts, nil
	}

	configured, err := uc.storage.ConfiguredCatalogs().Find(ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.In(lo.Map(targets, func(t PendingSchemaChangesTarget, _ int) uuid.UUID { return t.ConnectionID })...),
	})
	if err != nil {
		return nil, fmt.Errorf("finding configured catalogs: %w", err)
	}

	configuredByConn := lo.KeyBy(configured, func(c *pipelineservice.ConfiguredCatalog) uuid.UUID { return c.ConnectionID })
	latestBySource := make(map[uuid.UUID]*pipelineservice.CatalogDiscovery)

	for _, target := range targets {
		cfg, ok := configuredByConn[target.ConnectionID]
		if !ok {
			continue
		}

		latest, ok := latestBySource[target.SourceID]
		if !ok {
			latest, err = uc.latestDiscovery(ctx, target.SourceID)
			if err != nil {
				return nil, err
			}

			latestBySource[target.SourceID] = latest
		}

		if latest == nil {
			continue
		}

		changes, err := diffConfiguredCatalog(latest, cfg)
		if err != nil {
			return nil, fmt.Errorf("connection %s: %w", target.ConnectionID, err)
		}

		if len(changes) > 0 {
			counts[target.ConnectionID] = len(changes)
		}
	}

	return counts, nil
}

func (uc *CountPendingSchemaChanges) latestDiscovery(ctx context.Context, sourceID uuid.UUID) (*pipelineservice.CatalogDiscovery, error) {
	latest, err := uc.storage.CatalogDiscoveries().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(sourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		var nfe pipelineservice.NotFoundError
		if errors.As(err, &nfe) {
			return nil, nil
		}

		return nil, fmt.Errorf("getting latest catalog for source %s: %w", sourceID, err)
	}

	return latest, nil
}
