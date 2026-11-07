// Package pipelinegroups holds use cases that act on whole executor groups.
package pipelinegroups

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

var activeJobStatuses = []pipelineservice.JobStatus{
	pipelineservice.JobStatusScheduled,
	pipelineservice.JobStatusStarting,
	pipelineservice.JobStatusRunning,
}

// MoveExecutorGroup moves the sources and destinations of a workspace from
// one executor group to another.
type MoveExecutorGroup struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewMoveExecutorGroup creates a new MoveExecutorGroup use case.
func NewMoveExecutorGroup(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *MoveExecutorGroup {
	return &MoveExecutorGroup{storage: storage, audit: audit}
}

// MoveExecutorGroupParams holds parameters for moving an executor group.
type MoveExecutorGroupParams struct {
	WorkspaceID uuid.UUID
	FromGroup   string
	ToGroup     string
}

// MoveExecutorGroupResult counts the moved objects.
type MoveExecutorGroupResult struct {
	SourcesMoved      int
	DestinationsMoved int
}

// Execute moves every source and destination of the workspace in FromGroup to
// ToGroup in one transaction. It returns *pipelineservice.GroupBusyError and
// moves nothing while a job of a workspace connection is scheduled, starting
// or running in FromGroup, and a ValidationError when both name the same
// group.
func (uc *MoveExecutorGroup) Execute(ctx context.Context, params MoveExecutorGroupParams) (*MoveExecutorGroupResult, error) {
	from, err := pipelineservice.NormalizeExecutorGroup(params.FromGroup)
	if err != nil {
		return nil, err
	}

	to, err := pipelineservice.NormalizeExecutorGroup(params.ToGroup)
	if err != nil {
		return nil, err
	}

	if from == to {
		return nil, &pipelineservice.ValidationError{Field: "to_group", Message: "must differ from from_group"}
	}

	var (
		sources      []*pipelineservice.Source
		destinations []*pipelineservice.Destination
	)

	err = uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		sources, destinations, err = moveGroup(ctx, tx, params.WorkspaceID, from, to)

		return err
	})
	if err != nil {
		return nil, err
	}

	for _, src := range sources {
		uc.recordMove(ctx, params.WorkspaceID, "source", src.ID, src.Name, from, to)
	}

	for _, dest := range destinations {
		uc.recordMove(ctx, params.WorkspaceID, "destination", dest.ID, dest.Name, from, to)
	}

	return &MoveExecutorGroupResult{SourcesMoved: len(sources), DestinationsMoved: len(destinations)}, nil
}

func moveGroup(ctx context.Context, tx pipelineservice.Storage, workspaceID uuid.UUID, from, to string) ([]*pipelineservice.Source, []*pipelineservice.Destination, error) {
	sources, err := tx.Sources().Find(ctx, &pipelineservice.SourceFilter{
		WorkspaceID:   filter.Equals(workspaceID),
		ExecutorGroup: filter.Equals(from),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("listing sources of group %q: %w", from, err)
	}

	destinations, err := tx.Destinations().Find(ctx, &pipelineservice.DestinationFilter{
		WorkspaceID:   filter.Equals(workspaceID),
		ExecutorGroup: filter.Equals(from),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("listing destinations of group %q: %w", from, err)
	}

	if len(sources) == 0 && len(destinations) == 0 {
		return nil, nil, nil
	}

	if err := ensureNoActiveJobs(ctx, tx, workspaceID, from); err != nil {
		return nil, nil, err
	}

	now := time.Now()

	for _, src := range sources {
		src.ExecutorGroup = to
		src.UpdatedAt = now

		if _, err := tx.Sources().Update(ctx, src); err != nil {
			return nil, nil, fmt.Errorf("moving source %s: %w", src.ID, err)
		}
	}

	for _, dest := range destinations {
		dest.ExecutorGroup = to
		dest.UpdatedAt = now

		if _, err := tx.Destinations().Update(ctx, dest); err != nil {
			return nil, nil, fmt.Errorf("moving destination %s: %w", dest.ID, err)
		}
	}

	return sources, destinations, nil
}

func ensureNoActiveJobs(ctx context.Context, tx pipelineservice.Storage, workspaceID uuid.UUID, group string) error {
	connections, err := tx.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing connections: %w", err)
	}

	if len(connections) == 0 {
		return nil
	}

	connectionIDs := make([]uuid.UUID, len(connections))
	for i, conn := range connections {
		connectionIDs[i] = conn.ID
	}

	jobs, err := tx.Jobs().Find(ctx, &pipelineservice.JobFilter{
		ConnectionID:  filter.In(connectionIDs...),
		ExecutorGroup: filter.Equals(group),
		Status:        filter.In(activeJobStatuses...),
	})
	if err != nil {
		return fmt.Errorf("listing active jobs of group %q: %w", group, err)
	}

	if len(jobs) > 0 {
		return &pipelineservice.GroupBusyError{Group: group}
	}

	return nil
}

func (uc *MoveExecutorGroup) recordMove(ctx context.Context, workspaceID uuid.UUID, resourceType string, id uuid.UUID, name, from, to string) {
	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   workspaceID,
		Action:        "update",
		ResourceType:  resourceType,
		ResourceID:    id,
		ResourceLabel: name,
		Before:        map[string]any{"executor_group": from},
		After:         map[string]any{"executor_group": to},
	})
}
