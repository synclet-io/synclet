package pipelinegroups_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinegroups"
)

func TestMoveExecutorGroup(t *testing.T) {
	move := func(s *store, audit *captureAudit, ws uuid.UUID, from, to string) (*pipelinegroups.MoveExecutorGroupResult, error) {
		return pipelinegroups.NewMoveExecutorGroup(s, audit).Execute(context.Background(), pipelinegroups.MoveExecutorGroupParams{
			WorkspaceID: ws,
			FromGroup:   from,
			ToGroup:     to,
		})
	}

	t.Run("moves every source and destination of the workspace in the group", func(t *testing.T) {
		s := newStore()
		ws := uuid.New()
		src := s.addSource(ws, "a")
		dest := s.addDestination(ws, "a")
		s.addConnection(src, dest)
		idleSrc := s.addSource(ws, "a")
		otherGroup := s.addSource(ws, "c")
		otherGroupDest := s.addDestination(ws, "c")
		otherWorkspace := s.addDestination(uuid.New(), "a")
		audit := &captureAudit{}

		result, err := move(s, audit, ws, " a ", "b")
		require.NoError(t, err)

		assert.Equal(t, &pipelinegroups.MoveExecutorGroupResult{SourcesMoved: 2, DestinationsMoved: 1}, result)
		assert.Equal(t, "b", s.sources.records[src.ID].ExecutorGroup)
		assert.Equal(t, "b", s.sources.records[idleSrc.ID].ExecutorGroup)
		assert.Equal(t, "b", s.destinations.records[dest.ID].ExecutorGroup)
		assert.Equal(t, "c", s.sources.records[otherGroup.ID].ExecutorGroup)
		assert.Equal(t, "c", s.destinations.records[otherGroupDest.ID].ExecutorGroup)
		assert.Equal(t, "a", s.destinations.records[otherWorkspace.ID].ExecutorGroup)

		require.Len(t, audit.events, 3)

		for _, event := range audit.events {
			assert.Equal(t, "update", event.Action)
			assert.Equal(t, ws, event.WorkspaceID)
			assert.Equal(t, map[string]any{"executor_group": "a"}, event.Before)
			assert.Equal(t, map[string]any{"executor_group": "b"}, event.After)
		}
	})

	for _, status := range []pipelineservice.JobStatus{
		pipelineservice.JobStatusScheduled,
		pipelineservice.JobStatusStarting,
		pipelineservice.JobStatusRunning,
	} {
		t.Run("a "+status.String()+" job in the group blocks the move", func(t *testing.T) {
			s := newStore()
			ws := uuid.New()
			src := s.addSource(ws, "a")
			dest := s.addDestination(ws, "a")
			s.addJob(s.addConnection(src, dest), "a", status)

			audit := &captureAudit{}

			_, err := move(s, audit, ws, "a", "b")

			var busy *pipelineservice.GroupBusyError
			require.ErrorAs(t, err, &busy)
			assert.Equal(t, "a", busy.Group)
			assert.Equal(t, "a", s.sources.records[src.ID].ExecutorGroup)
			assert.Equal(t, "a", s.destinations.records[dest.ID].ExecutorGroup)
			assert.Empty(t, audit.events)
		})
	}

	t.Run("finished jobs and jobs of other groups do not block", func(t *testing.T) {
		s := newStore()
		ws := uuid.New()
		src := s.addSource(ws, "a")
		dest := s.addDestination(ws, "a")
		conn := s.addConnection(src, dest)
		s.addJob(conn, "a", pipelineservice.JobStatusCompleted)
		s.addJob(conn, "c", pipelineservice.JobStatusRunning)

		result, err := move(s, &captureAudit{}, ws, "a", "b")
		require.NoError(t, err)
		assert.Equal(t, 1, result.SourcesMoved)
		assert.Equal(t, "b", s.sources.records[src.ID].ExecutorGroup)
	})

	t.Run("another workspace's running job does not block", func(t *testing.T) {
		s := newStore()
		ws := uuid.New()
		src := s.addSource(ws, "a")
		otherSrc := s.addSource(uuid.New(), "a")
		otherDest := s.addDestination(otherSrc.WorkspaceID, "a")
		s.addJob(s.addConnection(otherSrc, otherDest), "a", pipelineservice.JobStatusRunning)

		_, err := move(s, &captureAudit{}, ws, "a", "b")
		require.NoError(t, err)
		assert.Equal(t, "b", s.sources.records[src.ID].ExecutorGroup)
	})

	t.Run("a failed destination update rolls the sources back", func(t *testing.T) {
		s := newStore()
		ws := uuid.New()
		src := s.addSource(ws, "a")
		dest := s.addDestination(ws, "a")
		s.destinations.updateErr = errors.New("db down")
		audit := &captureAudit{}

		_, err := move(s, audit, ws, "a", "b")
		require.ErrorIs(t, err, s.destinations.updateErr)
		assert.Equal(t, "a", s.sources.records[src.ID].ExecutorGroup)
		assert.Equal(t, "a", s.destinations.records[dest.ID].ExecutorGroup)
		assert.Empty(t, audit.events)
	})

	t.Run("nothing to move succeeds with zero counts", func(t *testing.T) {
		result, err := move(newStore(), &captureAudit{}, uuid.New(), "a", "b")
		require.NoError(t, err)
		assert.Equal(t, &pipelinegroups.MoveExecutorGroupResult{}, result)
	})

	t.Run("same group is a validation error", func(t *testing.T) {
		_, err := move(newStore(), &captureAudit{}, uuid.New(), "", "default")

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "to_group", validationErr.Field)
	})

	t.Run("invalid group is a validation error", func(t *testing.T) {
		_, err := move(newStore(), &captureAudit{}, uuid.New(), "a", "b\nc")

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}
