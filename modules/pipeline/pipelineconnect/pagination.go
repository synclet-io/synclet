package pipelineconnect

// paginateSlice applies offset-based pagination to a slice.
// If pageSize <= 0, returns all items (backward compatible).
// Returns the paginated slice and total count.
func paginateSlice[T any](items []T, pageSize, offset int32) (result []T, total int32) {
	total = int32(len(items))

	if pageSize <= 0 {
		return items, total
	}

	start := offset
	if start < 0 {
		start = 0
	}

	if start >= total {
		return nil, total
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return items[start:end], total
}
