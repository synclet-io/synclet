package pipelineservice

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// DefaultExecutorGroup is the group of an executor that names none, and of
// every source and destination created without one.
const DefaultExecutorGroup = "default"

// MaxExecutorGroupLength is the longest executor group name, in characters.
const MaxExecutorGroupLength = 100

// NormalizeExecutorGroup trims raw and maps an empty name to
// DefaultExecutorGroup. Names are case-sensitive labels kept verbatim; one
// longer than MaxExecutorGroupLength or holding a control character is a
// ValidationError on executor_group.
func NormalizeExecutorGroup(raw string) (string, error) {
	group := strings.TrimSpace(raw)
	if group == "" {
		return DefaultExecutorGroup, nil
	}

	if utf8.RuneCountInString(group) > MaxExecutorGroupLength {
		return "", &ValidationError{
			Field:   "executor_group",
			Message: fmt.Sprintf("must be at most %d characters", MaxExecutorGroupLength),
		}
	}

	if strings.ContainsFunc(group, unicode.IsControl) {
		return "", &ValidationError{Field: "executor_group", Message: "must not contain control characters"}
	}

	return group, nil
}

// ExecutorsBusyReason is the queue reason of a job whose group has live
// executors that have not claimed it.
func ExecutorsBusyReason(group string) string {
	return `executors in group "` + group + `" are busy`
}

// HasLiveExecutor reports whether an executor of group sent a registration
// heartbeat within silenceTimeout before now.
func HasLiveExecutor(ctx context.Context, storage Storage, group string, silenceTimeout time.Duration, now time.Time) (bool, error) {
	rows, err := storage.Executors().Find(ctx, &ExecutorFilter{ExecutorGroup: filter.Equals(group)})
	if err != nil {
		return false, fmt.Errorf("listing executors of group %q: %w", group, err)
	}

	cutoff := now.Add(-silenceTimeout)

	for _, row := range rows {
		if row.LastSeenAt.After(cutoff) {
			return true, nil
		}
	}

	return false, nil
}

// RequireLiveExecutor returns a NoLiveExecutorError when group has no live
// executor.
func RequireLiveExecutor(ctx context.Context, storage Storage, group string, silenceTimeout time.Duration, now time.Time) error {
	live, err := HasLiveExecutor(ctx, storage, group, silenceTimeout, now)
	if err != nil {
		return err
	}

	if !live {
		return &NoLiveExecutorError{Group: group}
	}

	return nil
}

// RequireSameExecutorGroup returns a ValidationError on executor_group when a
// connection's source and destination belong to different executor groups.
func RequireSameExecutorGroup(source *Source, destination *Destination) error {
	if source.ExecutorGroup == destination.ExecutorGroup {
		return nil
	}

	return &ValidationError{
		Field:   "executor_group",
		Message: `source group "` + source.ExecutorGroup + `" and destination group "` + destination.ExecutorGroup + `" differ`,
	}
}
