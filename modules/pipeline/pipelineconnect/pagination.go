package pipelineconnect

import "github.com/synclet-io/synclet/pkg/connectutil"

// normalizePageSize defaults non-positive sizes to connectutil.DefaultPageSize and caps the rest at connectutil.MaxPageSize.
func normalizePageSize(pageSize int32) int32 {
	if pageSize <= 0 {
		return connectutil.DefaultPageSize
	}

	return min(pageSize, connectutil.MaxPageSize)
}

// paginateSlice applies offset-based pagination to a slice.
// A pageSize <= 0 returns all items; larger sizes are capped at connectutil.MaxPageSize.
// Returns the paginated slice and total count.
func paginateSlice[T any](items []T, pageSize, offset int32) (result []T, total int32) {
	total = int32(len(items))

	if pageSize <= 0 {
		return items, total
	}

	pageSize = min(pageSize, connectutil.MaxPageSize)

	start := max(offset, 0)
	if start >= total {
		return nil, total
	}

	end := min(start+pageSize, total)

	return items[start:end], total
}
