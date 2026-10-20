package pipelineconnections

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

const (
	maxTagLength         = 32
	maxTagsPerConnection = 20
)

var tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.:-]*$`)

// NormalizeTags returns tags trimmed, lowercased, deduplicated and sorted. A tag
// is 1–32 characters matching ^[a-z0-9][a-z0-9_.:-]*$, and a connection holds at
// most 20 of them; any other input is a ValidationError on field "tags".
func NormalizeTags(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))

	for _, tag := range raw {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if len(normalized) > maxTagLength {
			return nil, &pipelineservice.ValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("invalid tag %q: longer than %d characters", tag, maxTagLength),
			}
		}

		if !tagPattern.MatchString(normalized) {
			return nil, &pipelineservice.ValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("invalid tag %q: must start with a letter or digit and contain only a-z, 0-9, '_', '.', ':' or '-'", tag),
			}
		}

		out = append(out, normalized)
	}

	slices.Sort(out)
	out = slices.Compact(out)

	if len(out) > maxTagsPerConnection {
		return nil, &pipelineservice.ValidationError{
			Field:   "tags",
			Message: fmt.Sprintf("too many tags: %d (max %d)", len(out), maxTagsPerConnection),
		}
	}

	return out, nil
}
