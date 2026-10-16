package pipelineconnect

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestMapErrorInUse(t *testing.T) {
	err := mapError(fmt.Errorf("deleting: %w", &pipelineservice.InUseError{
		Resource:        "source pg",
		ConnectionNames: []string{"orders"},
	}))

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "source pg is used by connections: orders")
}
