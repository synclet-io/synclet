package pipelineservice_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestNormalizeExecutorGroup(t *testing.T) {
	for _, tc := range []struct {
		name, raw, want string
	}{
		{name: "trims", raw: "  Prod Cluster ACME  ", want: "Prod Cluster ACME"},
		{name: "empty is default", raw: "", want: "default"},
		{name: "blank is default", raw: " \t ", want: "default"},
		{name: "case kept", raw: "EU", want: "EU"},
		{name: "100 characters", raw: strings.Repeat("ä", 100), want: strings.Repeat("ä", 100)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pipelineservice.NormalizeExecutorGroup(tc.raw)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	for _, tc := range []struct{ name, raw string }{
		{name: "101 characters", raw: strings.Repeat("a", 101)},
		{name: "control character", raw: "a\nb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pipelineservice.NormalizeExecutorGroup(tc.raw)

			var validationErr *pipelineservice.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, "executor_group", validationErr.Field)
		})
	}
}

func TestNoLiveExecutorError(t *testing.T) {
	require.EqualError(t, &pipelineservice.NoLiveExecutorError{Group: "Prod Cluster ACME"}, `no live executor in group "Prod Cluster ACME"`)
	assert.Equal(t, `executors in group "us" are busy`, pipelineservice.ExecutorsBusyReason("us"))
}
