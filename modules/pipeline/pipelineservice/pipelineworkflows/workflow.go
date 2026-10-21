// Package pipelineworkflows holds the use cases of workflows: named graphs of
// connection steps with a cron schedule and a history of runs.
package pipelineworkflows

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

// WorkflowDetail is a workflow with its steps and the edges between them.
type WorkflowDetail struct {
	Workflow     *pipelineservice.Workflow
	Steps        []*pipelineservice.WorkflowStep
	Dependencies []*pipelineservice.WorkflowStepDependency
}

// WorkflowRunDetail is a workflow run with its step runs.
type WorkflowRunDetail struct {
	Run      *pipelineservice.WorkflowRun
	StepRuns []*pipelineservice.WorkflowStepRun
}

func workflowToAuditPayload(detail *WorkflowDetail) map[string]any {
	if detail == nil {
		return nil
	}

	connByStep := make(map[uuid.UUID]uuid.UUID, len(detail.Steps))
	for _, step := range detail.Steps {
		connByStep[step.ID] = step.ConnectionID
	}

	after := make(map[uuid.UUID][]string, len(detail.Steps))
	for _, dep := range detail.Dependencies {
		after[dep.StepID] = append(after[dep.StepID], connByStep[dep.UpstreamStepID].String())
	}

	steps := make([]map[string]any, 0, len(detail.Steps))
	for _, step := range detail.Steps {
		steps = append(steps, map[string]any{
			"connection_id":        step.ConnectionID.String(),
			"after_connection_ids": append([]string{}, after[step.ID]...),
		})
	}

	payload := map[string]any{"name": detail.Workflow.Name, "steps": steps}
	if detail.Workflow.Schedule != nil {
		payload["schedule"] = *detail.Workflow.Schedule
	}

	return payload
}

// validateDefinition checks a workflow's name, schedule and graph, and that
// every step is a connection of the workspace with configured streams.
func validateDefinition(
	ctx context.Context,
	tx pipelineservice.Storage,
	workspaceID uuid.UUID,
	name string,
	schedule *string,
	steps []pipelineservice.WorkflowGraphStep,
) error {
	if strings.TrimSpace(name) == "" {
		return &pipelineservice.ValidationError{Field: "name", Message: "name is required"}
	}

	if schedule != nil && *schedule != "" {
		if _, err := pipelineservice.CronParser.Parse(*schedule); err != nil {
			return &pipelineservice.ValidationError{Field: "schedule", Message: "invalid cron expression: " + err.Error()}
		}
	}

	if err := pipelineservice.ValidateWorkflowGraph(steps); err != nil {
		return err
	}

	ids := make([]uuid.UUID, 0, len(steps))
	for _, step := range steps {
		ids = append(ids, step.ConnectionID)
	}

	conns, err := tx.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.In(ids...),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("loading step connections: %w", err)
	}

	found := make(map[uuid.UUID]bool, len(conns))
	for _, conn := range conns {
		found[conn.ID] = true
	}

	for _, id := range ids {
		if !found[id] {
			return &pipelineservice.ValidationError{Field: "steps", Message: id.String() + " is not a connection of this workspace"}
		}
	}

	var noStreams *pipelineservice.NoConfiguredStreamsError

	err = pipelineservice.RequireConfiguredStreams(ctx, tx, conns...)
	if errors.As(err, &noStreams) {
		return &pipelineservice.ValidationError{Field: "steps", Message: noStreams.Error()}
	}

	return err
}

// requireRunnableSteps returns a *NoConfiguredStreamsError when a connection
// of the given steps has no configured stream.
func requireRunnableSteps(ctx context.Context, tx pipelineservice.Storage, steps []*pipelineservice.WorkflowStep) error {
	if len(steps) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(steps))
	for i, step := range steps {
		ids[i] = step.ConnectionID
	}

	conns, err := tx.Connections().Find(ctx, &pipelineservice.ConnectionFilter{ID: filter.In(ids...)})
	if err != nil {
		return fmt.Errorf("loading step connections: %w", err)
	}

	return pipelineservice.RequireConfiguredStreams(ctx, tx, conns...)
}

func normalizeSchedule(schedule *string) *string {
	if schedule == nil || strings.TrimSpace(*schedule) == "" {
		return nil
	}

	trimmed := strings.TrimSpace(*schedule)

	return &trimmed
}

// saveSteps makes the workflow's steps match the definition: it keeps the
// step of each connection that stays, removes the others, adds the new ones
// and replaces every edge.
func saveSteps(ctx context.Context, tx pipelineservice.Storage, workflowID uuid.UUID, steps []pipelineservice.WorkflowGraphStep) error {
	existing, err := tx.WorkflowSteps().Find(ctx, &pipelineservice.WorkflowStepFilter{WorkflowID: filter.Equals(workflowID)})
	if err != nil {
		return fmt.Errorf("listing steps: %w", err)
	}

	stepByConn := make(map[uuid.UUID]uuid.UUID, len(existing))
	for _, step := range existing {
		stepByConn[step.ConnectionID] = step.ID
	}

	wanted := make(map[uuid.UUID]bool, len(steps))
	for _, step := range steps {
		wanted[step.ConnectionID] = true
	}

	if err := tx.WorkflowStepDependencies().Delete(ctx, &pipelineservice.WorkflowStepDependencyFilter{
		WorkflowID: filter.Equals(workflowID),
	}); err != nil {
		return fmt.Errorf("deleting step dependencies: %w", err)
	}

	for connID, stepID := range stepByConn {
		if wanted[connID] {
			continue
		}

		if err := tx.WorkflowSteps().Delete(ctx, &pipelineservice.WorkflowStepFilter{ID: filter.Equals(stepID)}); err != nil {
			return fmt.Errorf("deleting step %s: %w", stepID, err)
		}

		delete(stepByConn, connID)
	}

	for _, step := range steps {
		if _, ok := stepByConn[step.ConnectionID]; ok {
			continue
		}

		created, err := tx.WorkflowSteps().Create(ctx, &pipelineservice.WorkflowStep{
			ID: uuid.New(), WorkflowID: workflowID, ConnectionID: step.ConnectionID,
		})
		if err != nil {
			return fmt.Errorf("creating step for connection %s: %w", step.ConnectionID, err)
		}

		stepByConn[step.ConnectionID] = created.ID
	}

	var deps []*pipelineservice.WorkflowStepDependency

	for _, step := range steps {
		seen := map[uuid.UUID]bool{}

		for _, upstream := range step.AfterConnectionIDs {
			if seen[upstream] {
				continue
			}

			seen[upstream] = true
			deps = append(deps, &pipelineservice.WorkflowStepDependency{
				ID:             uuid.New(),
				WorkflowID:     workflowID,
				StepID:         stepByConn[step.ConnectionID],
				UpstreamStepID: stepByConn[upstream],
			})
		}
	}

	if len(deps) > 0 {
		if _, err := tx.WorkflowStepDependencies().BatchCreate(ctx, deps); err != nil {
			return fmt.Errorf("creating step dependencies: %w", err)
		}
	}

	return nil
}

// handOverSchedules turns off the own schedule of every step's connection:
// a connection in a workflow is run by the workflow. It locks the connection
// rows, so a concurrent UpdateConnection either commits first or sees the step.
func handOverSchedules(ctx context.Context, tx pipelineservice.Storage, workspaceID uuid.UUID, steps []pipelineservice.WorkflowGraphStep) error {
	if len(steps) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(steps))
	for _, step := range steps {
		ids = append(ids, step.ConnectionID)
	}

	conns, err := tx.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.In(ids...),
		WorkspaceID: filter.Equals(workspaceID),
	}, dbutil.WithForUpdate())
	if err != nil {
		return fmt.Errorf("locking step connections: %w", err)
	}

	for _, conn := range conns {
		if conn.Schedule == nil && conn.NextScheduledAt == nil {
			continue
		}

		conn.Schedule = nil
		conn.NextScheduledAt = nil
		conn.UpdatedAt = time.Now()

		if _, err := tx.Connections().Update(ctx, conn); err != nil {
			return fmt.Errorf("turning off schedule of connection %s: %w", conn.ID, err)
		}
	}

	return nil
}

func getWorkflow(ctx context.Context, storage pipelineservice.Storage, id, workspaceID uuid.UUID) (*pipelineservice.Workflow, error) {
	workflow, err := storage.Workflows().First(ctx, &pipelineservice.WorkflowFilter{
		ID:          filter.Equals(id),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting workflow: %w", err)
	}

	return workflow, nil
}

func loadDetails(ctx context.Context, storage pipelineservice.Storage, workflows []*pipelineservice.Workflow) ([]*WorkflowDetail, error) {
	if len(workflows) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(workflows))
	details := make(map[uuid.UUID]*WorkflowDetail, len(workflows))
	result := make([]*WorkflowDetail, 0, len(workflows))

	for _, workflow := range workflows {
		ids = append(ids, workflow.ID)
		detail := &WorkflowDetail{Workflow: workflow}
		details[workflow.ID] = detail
		result = append(result, detail)
	}

	steps, err := storage.WorkflowSteps().Find(ctx, &pipelineservice.WorkflowStepFilter{WorkflowID: filter.In(ids...)})
	if err != nil {
		return nil, fmt.Errorf("listing workflow steps: %w", err)
	}

	for _, step := range steps {
		details[step.WorkflowID].Steps = append(details[step.WorkflowID].Steps, step)
	}

	deps, err := storage.WorkflowStepDependencies().Find(ctx, &pipelineservice.WorkflowStepDependencyFilter{WorkflowID: filter.In(ids...)})
	if err != nil {
		return nil, fmt.Errorf("listing workflow step dependencies: %w", err)
	}

	for _, dep := range deps {
		details[dep.WorkflowID].Dependencies = append(details[dep.WorkflowID].Dependencies, dep)
	}

	return result, nil
}

func loadDetail(ctx context.Context, storage pipelineservice.Storage, workflow *pipelineservice.Workflow) (*WorkflowDetail, error) {
	details, err := loadDetails(ctx, storage, []*pipelineservice.Workflow{workflow})
	if err != nil {
		return nil, err
	}

	return details[0], nil
}

func hasActiveRun(ctx context.Context, tx pipelineservice.Storage, workflowID uuid.UUID) (bool, error) {
	count, err := tx.WorkflowRuns().Count(ctx, &pipelineservice.WorkflowRunFilter{
		WorkflowID: filter.Equals(workflowID),
		Status:     filter.Equals(pipelineservice.WorkflowRunStatusRunning),
	})
	if err != nil {
		return false, fmt.Errorf("counting active runs: %w", err)
	}

	return count > 0, nil
}

func loadRunDetail(ctx context.Context, storage pipelineservice.Storage, runID uuid.UUID) (*WorkflowRunDetail, error) {
	run, err := storage.WorkflowRuns().First(ctx, &pipelineservice.WorkflowRunFilter{ID: filter.Equals(runID)})
	if err != nil {
		return nil, fmt.Errorf("getting workflow run: %w", err)
	}

	stepRuns, err := storage.WorkflowStepRuns().Find(ctx, &pipelineservice.WorkflowStepRunFilter{RunID: filter.Equals(runID)})
	if err != nil {
		return nil, fmt.Errorf("listing step runs: %w", err)
	}

	return &WorkflowRunDetail{Run: run, StepRuns: stepRuns}, nil
}

// lockWorkflowRuns serializes run starts, edits and the advancer on the
// scheduler lock.
func lockWorkflowRuns(ctx context.Context, tx pipelineservice.Storage) error {
	if err := tx.WithAdvisoryLock(ctx, pipelinejobs.SchedulerLockScope, 1); err != nil {
		return fmt.Errorf("acquiring scheduler lock: %w", err)
	}

	return nil
}

// sendRunStarted sends the workflow_run.started event that hands the run to
// the jobs process for advancing.
func sendRunStarted(ctx context.Context, tx pipelineservice.Storage, workspaceID uuid.UUID, run *pipelineservice.WorkflowRun) error {
	if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeWorkflowRunStarted,
		Data: &pipelineservice.WorkflowRunStartedEventData{
			WorkspaceID: workspaceID,
			WorkflowID:  run.WorkflowID,
			RunID:       run.ID,
		},
	}); err != nil {
		return fmt.Errorf("publishing workflow_run.started event for run %s: %w", run.ID, err)
	}

	return nil
}
