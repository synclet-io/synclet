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

func TestMapErrorValidation(t *testing.T) {
	err := mapError(fmt.Errorf("creating: %w", &pipelineservice.ValidationError{Field: "schedule", Message: "bad cron"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "bad cron")
}

func TestMapErrorNoCachedSpec(t *testing.T) {
	err := mapError(fmt.Errorf("spec: %w", pipelineservice.ErrNoCachedSpec))

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestMapErrorCatalogNotReady(t *testing.T) {
	err := mapError(pipelineservice.ErrCatalogNotReady)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "catalog is not discovered yet; retry")
}

func TestMapErrorNoConfiguredStreams(t *testing.T) {
	err := mapError(fmt.Errorf("queuing job: %w", &pipelineservice.NoConfiguredStreamsError{ConnectionNames: []string{"pg-to-bq"}}))

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.Contains(t, err.Error(), `connection "pg-to-bq" has no configured streams`)
}
