package executorgroups_test

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

type noopAudit struct{}

func (noopAudit) Record(context.Context, pipelineservice.AuditEvent) {}

var errDestinationUpdate = errors.New("destination update failed")

// failingDestinationUpdates fails every destination update made inside a
// transaction, after the writes before it went through.
type failingDestinationUpdates struct {
	pipelineservice.Storage
}

func (s failingDestinationUpdates) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return s.Storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		return cb(ctx, failingDestinationTx{Storage: tx})
	})
}

type failingDestinationTx struct {
	pipelineservice.Storage
}

func (s failingDestinationTx) Destinations() pipelineservice.DestinationsStorage {
	return failingDestinations{DestinationsStorage: s.Storage.Destinations()}
}

type failingDestinations struct {
	pipelineservice.DestinationsStorage
}

func (failingDestinations) Update(context.Context, *pipelineservice.Destination) (*pipelineservice.Destination, error) {
	return nil, errDestinationUpdate
}

func (f fixture) groups(t *testing.T) (source, destination string) {
	t.Helper()

	require.NoError(t, f.db.Raw(`SELECT executor_group FROM pipeline.sources WHERE id = ?`, f.sourceID).Scan(&source).Error)
	require.NoError(t, f.db.Raw(`SELECT executor_group FROM pipeline.destinations WHERE id = ?`, f.destID).Scan(&destination).Error)

	return source, destination
}

// TestMoveExecutorGroup asserts that a running job in the group blocks the
// move and leaves both rows in place, and that without it the source and the
// destination both land in the target group.
func TestMoveExecutorGroup(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	from, to := "a-"+uuid.NewString(), "b-"+uuid.NewString()

	require.NoError(t, f.db.Exec(`UPDATE pipeline.sources SET executor_group = ? WHERE id = ?`, from, f.sourceID).Error)
	require.NoError(t, f.db.Exec(`UPDATE pipeline.destinations SET executor_group = ? WHERE id = ?`, from, f.destID).Error)

	jobID := f.scheduleJob(t, from)
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET status = 'running' WHERE id = ?`, jobID).Error)

	move := pipelinegroups.NewMoveExecutorGroup(f.storage, noopAudit{})
	params := pipelinegroups.MoveExecutorGroupParams{WorkspaceID: f.workspaceID, FromGroup: from, ToGroup: to}

	_, err := move.Execute(ctx, params)

	var busy *pipelineservice.GroupBusyError
	require.ErrorAs(t, err, &busy)

	source, destination := f.groups(t)
	assert.Equal(t, from, source)
	assert.Equal(t, from, destination)

	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET status = 'completed' WHERE id = ?`, jobID).Error)

	result, err := move.Execute(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, &pipelinegroups.MoveExecutorGroupResult{SourcesMoved: 1, DestinationsMoved: 1}, result)

	source, destination = f.groups(t)
	assert.Equal(t, to, source)
	assert.Equal(t, to, destination)
}

// TestMoveExecutorGroupRollsBack asserts that a destination update failing
// after the source was moved leaves the source in the original group.
func TestMoveExecutorGroupRollsBack(t *testing.T) {
	f := setup(t)
	from, to := "a-"+uuid.NewString(), "b-"+uuid.NewString()

	require.NoError(t, f.db.Exec(`UPDATE pipeline.sources SET executor_group = ? WHERE id = ?`, from, f.sourceID).Error)
	require.NoError(t, f.db.Exec(`UPDATE pipeline.destinations SET executor_group = ? WHERE id = ?`, from, f.destID).Error)

	_, err := pipelinegroups.NewMoveExecutorGroup(failingDestinationUpdates{Storage: f.storage}, noopAudit{}).Execute(context.Background(), pipelinegroups.MoveExecutorGroupParams{
		WorkspaceID: f.workspaceID,
		FromGroup:   from,
		ToGroup:     to,
	})
	require.ErrorIs(t, err, errDestinationUpdate)

	source, destination := f.groups(t)
	assert.Equal(t, from, source)
	assert.Equal(t, from, destination)
}
