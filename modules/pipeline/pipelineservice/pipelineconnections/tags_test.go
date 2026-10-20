package pipelineconnections_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

func TestNormalizeTags(t *testing.T) {
	tooMany := make([]string, 0, 21)
	for i := range 21 {
		tooMany = append(tooMany, fmt.Sprintf("tag%d", i))
	}

	maxCount := make([]string, 0, 20)
	for i := range 20 {
		maxCount = append(maxCount, fmt.Sprintf("tag%02d", i))
	}

	tests := []struct {
		name    string
		in      []string
		want    []string
		wantErr string
	}{
		{name: "nil is empty", in: nil, want: []string{}},
		{name: "trims and lowercases", in: []string{" Finance "}, want: []string{"finance"}},
		{name: "collapses duplicates", in: []string{"prod", "PROD", "prod "}, want: []string{"prod"}},
		{name: "sorts", in: []string{"prod", "finance", "team-data"}, want: []string{"finance", "prod", "team-data"}},
		{name: "allows the key:value convention", in: []string{"team:data"}, want: []string{"team:data"}},
		{name: "allows dot and underscore", in: []string{"v1.2_x"}, want: []string{"v1.2_x"}},
		{name: "allows 32 chars", in: []string{strings.Repeat("a", 32)}, want: []string{strings.Repeat("a", 32)}},
		{name: "allows 20 tags", in: maxCount, want: maxCount},
		{name: "rejects empty", in: []string{""}, wantErr: `invalid tag ""`},
		{name: "rejects blank", in: []string{"  "}, wantErr: `invalid tag "  "`},
		{name: "rejects 33 chars", in: []string{strings.Repeat("a", 33)}, wantErr: strings.Repeat("a", 33)},
		{name: "rejects a space inside", in: []string{"Fin ance"}, wantErr: `invalid tag "Fin ance"`},
		{name: "rejects non-ascii", in: []string{"ünï"}, wantErr: `invalid tag "ünï"`},
		{name: "rejects a leading dash", in: []string{"-x"}, wantErr: `invalid tag "-x"`},
		{name: "rejects more than 20 tags", in: tooMany, wantErr: "too many tags: 21 (max 20)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pipelineconnections.NormalizeTags(tc.in)
			if tc.wantErr != "" {
				var validationErr *pipelineservice.ValidationError
				require.ErrorAs(t, err, &validationErr)
				assert.Equal(t, "tags", validationErr.Field)
				assert.Contains(t, err.Error(), tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
