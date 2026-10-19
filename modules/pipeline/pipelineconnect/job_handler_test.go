package pipelineconnect

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

func TestNormalizePageSize(t *testing.T) {
	assert.Equal(t, int32(connectutil.DefaultPageSize), normalizePageSize(0))
	assert.Equal(t, int32(connectutil.DefaultPageSize), normalizePageSize(-5))
	assert.Equal(t, int32(10), normalizePageSize(10))
	assert.Equal(t, int32(connectutil.MaxPageSize), normalizePageSize(100_000))
}

func TestPaginateSlice_CapsPageSize(t *testing.T) {
	items := make([]int, connectutil.MaxPageSize*2)

	page, total := paginateSlice(items, connectutil.MaxPageSize*2, 0)
	assert.Len(t, page, connectutil.MaxPageSize)
	assert.Equal(t, int32(connectutil.MaxPageSize*2), total)

	all, _ := paginateSlice(items, 0, 0)
	assert.Len(t, all, connectutil.MaxPageSize*2)
}

func TestJobStatusesFromProto(t *testing.T) {
	statuses, err := jobStatusesFromProto([]pipelinev1.JobStatus{
		pipelinev1.JobStatus_JOB_STATUS_RUNNING,
		pipelinev1.JobStatus_JOB_STATUS_STARTING,
		pipelinev1.JobStatus_JOB_STATUS_FAILED,
	})
	require.NoError(t, err)
	assert.Equal(t, []pipelineservice.JobStatus{
		pipelineservice.JobStatusRunning,
		pipelineservice.JobStatusStarting,
		pipelineservice.JobStatusFailed,
	}, statuses)

	empty, err := jobStatusesFromProto(nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	_, err = jobStatusesFromProto([]pipelinev1.JobStatus{pipelinev1.JobStatus_JOB_STATUS_UNSPECIFIED})
	require.Error(t, err)
}

func TestChunkSender_SplitsLargeWrites(t *testing.T) {
	var chunks [][]byte

	sender := &chunkSender{send: func(chunk []byte) error {
		chunks = append(chunks, chunk)

		return nil
	}}

	data := bytes.Repeat([]byte("x"), logChunkSize*2+5)
	n, err := sender.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	require.Len(t, chunks, 3)
	assert.Len(t, chunks[0], logChunkSize)
	assert.Len(t, chunks[2], 5)
	assert.Equal(t, data, bytes.Join(chunks, nil))
}
