package pipelineconnect

import (
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/samber/lo"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// mapError maps pipeline domain errors to ConnectRPC error codes.
// Unknown errors are returned raw so the error interceptor can log them.
func mapError(err error) error {
	if notFoundErr, ok := lo.ErrorsAs[pipelineservice.NotFoundError](err); ok {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("%s not found", strings.ToLower(string(notFoundErr))))
	}

	if alreadyExists, ok := lo.ErrorsAs[pipelineservice.AlreadyExistsError](err); ok {
		return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("%s already exists", strings.ToLower(string(alreadyExists))))
	}

	if validationErr, ok := lo.ErrorsAs[*pipelineservice.ValidationError](err); ok {
		return connect.NewError(connect.CodeInvalidArgument, validationErr)
	}

	if errors.Is(err, pipelineservice.ErrStateDataInvalidJSON) ||
		errors.Is(err, pipelineservice.ErrMissingCheckTaskParams) ||
		errors.Is(err, pipelineservice.ErrEmptyFieldPath) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	if inUseErr, ok := lo.ErrorsAs[*pipelineservice.InUseError](err); ok {
		return connect.NewError(connect.CodeFailedPrecondition, inUseErr)
	}

	if noStreamsErr, ok := lo.ErrorsAs[*pipelineservice.NoConfiguredStreamsError](err); ok {
		return connect.NewError(connect.CodeFailedPrecondition, noStreamsErr)
	}

	if errors.Is(err, pipelineservice.ErrConnectorNotLinked) || errors.Is(err, pipelineservice.ErrNoCachedSpec) ||
		errors.Is(err, pipelineservice.ErrWorkflowRunActive) || errors.Is(err, pipelineservice.ErrWorkflowRunNotFailed) ||
		errors.Is(err, pipelineservice.ErrConnectionInWorkflow) || errors.Is(err, pipelineservice.ErrCatalogNotReady) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}

	return err
}
