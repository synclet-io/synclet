package pipelinejobs

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListExecutors returns the executors that sent a registration heartbeat
// within the silence timeout. Rows of silent executors stay in storage.
type ListExecutors struct {
	storage        pipelineservice.Storage
	silenceTimeout time.Duration
	now            func() time.Time
}

// NewListExecutors creates a new ListExecutors use case.
func NewListExecutors(storage pipelineservice.Storage, cfg pipelineservice.Config) *ListExecutors {
	return &ListExecutors{storage: storage, silenceTimeout: cfg.ExecutorSilenceTimeout, now: time.Now}
}

// Execute returns the live executors.
func (uc *ListExecutors) Execute(ctx context.Context) ([]*pipelineservice.Executor, error) {
	rows, err := uc.storage.Executors().Find(ctx, &pipelineservice.ExecutorFilter{})
	if err != nil {
		return nil, fmt.Errorf("listing executors: %w", err)
	}

	cutoff := uc.now().Add(-uc.silenceTimeout)
	live := make([]*pipelineservice.Executor, 0, len(rows))

	for _, row := range rows {
		if row.LastSeenAt.After(cutoff) {
			live = append(live, row)
		}
	}

	sort.Slice(live, func(i, j int) bool { return live[i].InstanceID < live[j].InstanceID })

	return live, nil
}
