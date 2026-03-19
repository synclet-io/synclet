package pipelineconnect

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// mapError maps pipeline domain errors to ConnectRPC error codes.
// Unknown errors are returned raw so the error interceptor can log them.
func mapError(err error) error {
	var notFound pipelineservice.NotFoundError
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var alreadyExists pipelineservice.AlreadyExistsError
	if errors.As(err, &alreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	var validation *pipelineservice.ValidationError
	if errors.As(err, &validation) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return err
}
