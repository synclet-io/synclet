package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestrateCommand_RuntimeErrorPrintsNoUsage(t *testing.T) {
	cmd := newOrchestrateCommand()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--task-mode"})

	err := cmd.Execute()

	require.ErrorContains(t, err, "--task-id is required")
	assert.NotContains(t, out.String(), "Usage:")
	assert.NotContains(t, out.String(), "Error:")
}
