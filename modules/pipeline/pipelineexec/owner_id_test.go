package pipelineexec

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOwnerID(t *testing.T) {
	const (
		setting       = "K8S_EXECUTOR_OWNER_ID"
		prefixSetting = "K8S_EXECUTOR_OWNER_ID_PREFIX"
	)

	podHostname := func() (string, error) { return "t-synclet-executor-2", nil }

	tests := []struct {
		name         string
		id           string
		prefix       string
		hostname     func() (string, error)
		want         string
		wantErrNames []string
	}{
		{name: "nothing set", hostname: podHostname, wantErrNames: []string{setting, "Kubernetes namespace"}},
		{name: "id set", id: "blue", hostname: podHostname, want: "blue"},
		{name: "id set without hostname", id: "blue", hostname: func() (string, error) { return "", errors.New("no hostname") }, want: "blue"},
		{name: "prefix with ordinal hostname", prefix: "blue", hostname: podHostname, want: "blue-2"},
		{name: "prefix with hostname without ordinal", prefix: "blue", hostname: func() (string, error) { return "myhost", nil }, wantErrNames: []string{prefixSetting}},
		{name: "prefix with hostname ending in dash", prefix: "blue", hostname: func() (string, error) { return "myhost-", nil }, wantErrNames: []string{prefixSetting}},
		{name: "prefix without hostname", prefix: "blue", hostname: func() (string, error) { return "", errors.New("no hostname") }, wantErrNames: []string{prefixSetting}},
		{name: "id and prefix set", id: "blue", prefix: "blue", hostname: podHostname, wantErrNames: []string{setting, prefixSetting}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveOwnerID(tt.id, tt.prefix, setting, prefixSetting, "Kubernetes namespace", tt.hostname)
			if tt.wantErrNames != nil {
				for _, name := range tt.wantErrNames {
					require.ErrorContains(t, err, name)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := ResolveOwnerID("", "", "DOCKER_EXECUTOR_OWNER_ID", "", "Docker daemon", podHostname)
	require.ErrorContains(t, err, "DOCKER_EXECUTOR_OWNER_ID")
	require.ErrorContains(t, err, "Docker daemon")
	assert.NotContains(t, err.Error(), "Kubernetes")
}
