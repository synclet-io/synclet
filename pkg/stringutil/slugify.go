package stringutil

import (
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-friendly slug.
func Slugify(s string) string {
	slug := strings.ToLower(strings.TrimSpace(s))
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	return slug
}
