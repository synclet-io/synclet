package pipelinejobs

import (
	"sort"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// LatestExecutorPerType returns, for each executor type present, the most
// recently seen executor of that type, ordered by type.
func LatestExecutorPerType(executors []*pipelineservice.Executor) []*pipelineservice.Executor {
	latest := make(map[pipelineservice.ExecutorType]*pipelineservice.Executor)

	for _, executor := range executors {
		if current, ok := latest[executor.Type]; !ok || executor.LastSeenAt.After(current.LastSeenAt) {
			latest[executor.Type] = executor
		}
	}

	result := make([]*pipelineservice.Executor, 0, len(latest))
	for _, executor := range latest {
		result = append(result, executor)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Type < result[j].Type })

	return result
}
