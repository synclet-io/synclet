package pipelineexec

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOwnerID(t *testing.T) {
	hostname := func() (string, error) { return "executor-1", nil }
	noHostname := func() (string, error) { return "", errors.New("no hostname") }

	ownerID, err := ResolveOwnerID("", "K8S_EXECUTOR_OWNER_ID", hostname)
	require.NoError(t, err)
	assert.Equal(t, "executor-1", ownerID)

	ownerID, err = ResolveOwnerID("blue", "K8S_EXECUTOR_OWNER_ID", noHostname)
	require.NoError(t, err)
	assert.Equal(t, "blue", ownerID)

	_, err = ResolveOwnerID("", "DOCKER_EXECUTOR_OWNER_ID", noHostname)
	require.ErrorContains(t, err, "DOCKER_EXECUTOR_OWNER_ID")

	_, err = ResolveOwnerID("", "K8S_EXECUTOR_OWNER_ID", func() (string, error) { return "", nil })
	require.ErrorContains(t, err, "K8S_EXECUTOR_OWNER_ID")
}
