package pipelinecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// RecheckStreamMappers compiles the mappers of every connection on a source
// against a new discovery, and pauses each connection whose mappers no longer
// compile, whatever its schema change policy, with a reason naming the stream
// and the step.
type RecheckStreamMappers struct {
	storage pipelineservice.Storage
}

// NewRecheckStreamMappers creates a new RecheckStreamMappers use case.
func NewRecheckStreamMappers(storage pipelineservice.Storage) *RecheckStreamMappers {
	return &RecheckStreamMappers{storage: storage}
}

// Execute rechecks the connections on sourceID that are not paused against catalog.
func (uc *RecheckStreamMappers) Execute(ctx context.Context, sourceID uuid.UUID, catalog *protocol.AirbyteCatalog) error {
	conns, err := uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		SourceID: filter.Equals(sourceID),
	})
	if err != nil {
		return fmt.Errorf("listing connections of source: %w", err)
	}

	discovered := streamsByKey(catalog.Streams)

	var errs []error

	for _, conn := range conns {
		if conn.Status == pipelineservice.ConnectionStatusPaused {
			continue
		}

		if err := uc.recheck(ctx, conn, discovered); err != nil {
			errs = append(errs, fmt.Errorf("connection %s: %w", conn.ID, err))
		}
	}

	return errors.Join(errs...)
}

func (uc *RecheckStreamMappers) recheck(ctx context.Context, conn *pipelineservice.Connection, discovered map[string]protocol.AirbyteStream) error {
	record, err := uc.storage.ConfiguredCatalogs().First(ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.Equals(conn.ID),
	})
	if err != nil {
		var nfe pipelineservice.NotFoundError
		if errors.As(err, &nfe) {
			return nil
		}

		return fmt.Errorf("getting configured catalog: %w", err)
	}

	var streams []protocol.ConfiguredAirbyteStream
	if err := json.Unmarshal([]byte(record.StreamsJSON), &streams); err != nil {
		return fmt.Errorf("unmarshaling configured streams: %w", err)
	}

	reason := brokenMappersReason(streams, discovered)
	if reason == "" {
		return nil
	}

	now := time.Now()
	conn.Status = pipelineservice.ConnectionStatusPaused
	conn.StatusReason = &reason
	conn.UpdatedAt = now
	pipelineservice.RecomputeNextScheduledAt(conn, now)

	if _, err := uc.storage.Connections().Update(ctx, conn); err != nil {
		return fmt.Errorf("pausing connection: %w", err)
	}

	return nil
}

// brokenMappersReason returns why the first stream whose mappers do not
// compile against its discovered schema fails, or "" when all compile.
func brokenMappersReason(streams []protocol.ConfiguredAirbyteStream, discovered map[string]protocol.AirbyteStream) string {
	for i := range streams {
		stream := &streams[i]
		if !pipelineservice.HasMappers(stream) {
			continue
		}

		key := streamKey(stream.Stream.Namespace, stream.Stream.Name)

		available, ok := discovered[key]
		if !ok {
			continue
		}

		_, result, err := pipelineservice.CompileStreamMappers(stream, available.JSONSchema, nil)
		if err != nil {
			return fmt.Sprintf("stream %q: %v", key, err)
		}

		if len(result.Errors) > 0 {
			return fmt.Sprintf("stream %q: mapper %s", key, pipelineservice.FormatMapperIssue(result.Errors[0]))
		}
	}

	return ""
}
