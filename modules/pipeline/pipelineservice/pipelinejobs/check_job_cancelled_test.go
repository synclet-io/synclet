package pipelinejobs

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type cancelCheckStorage struct {
	pipelineservice.Storage
	jobs *cancelCheckJobs
}

func (s *cancelCheckStorage) Jobs() pipelineservice.JobsStorage { return s.jobs }

type cancelCheckJobs struct {
	pipelineservice.JobsStorage
	job *pipelineservice.Job
	err error
}

func (j *cancelCheckJobs) First(context.Context, *pipelineservice.JobFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Job, error) {
	return j.job, j.err
}

func TestCheckJobCancelled(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name      string
		jobs      *cancelCheckJobs
		cancelled bool
		wantErr   bool
	}{
		{name: "running job", jobs: &cancelCheckJobs{job: &pipelineservice.Job{Status: pipelineservice.JobStatusRunning}}},
		{name: "cancelled job", jobs: &cancelCheckJobs{job: &pipelineservice.Job{Status: pipelineservice.JobStatusCancelled}}, cancelled: true},
		{name: "deleted job", jobs: &cancelCheckJobs{err: errors.Join(pipelineservice.ErrJobNotFound, errors.New("record not found"))}, cancelled: true},
		{name: "storage failure", jobs: &cancelCheckJobs{err: errors.New("db down")}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cancelled, err := NewCheckJobCancelled(&cancelCheckStorage{jobs: tc.jobs}).Execute(ctx, CheckJobCancelledParams{JobID: uuid.New()})

			if tc.wantErr {
				require.Error(t, err)
				assert.False(t, cancelled)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.cancelled, cancelled)
		})
	}
}
