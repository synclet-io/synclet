package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Hello World", "hello-world"},
		{"  trim spaces  ", "trim-spaces"},
		{"MIXED-Case_Underscores", "mixed-case-underscores"},
		{"Multiple   spaces", "multiple-spaces"},
		{"already-a-slug", "already-a-slug"},
		{"With Numbers 123", "with-numbers-123"},
		{"!!leading-and-trailing!!", "leading-and-trailing"},
		{"unicode-stripped-é-ü", "unicode-stripped"},
		{"", ""},
		{"   ", ""},
		{"---", ""},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, Slugify(tc.in))
		})
	}
}
