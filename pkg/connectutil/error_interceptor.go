package connectutil

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
)

// errorClassifier is the interceptor that catches unhandled errors from handlers.
type errorClassifier struct {
	logger *slog.Logger
}

// NewErrorInterceptor creates a ConnectRPC interceptor that:
// - Passes through *connect.Error as-is (handlers classify domain errors explicitly)
// - Logs unexpected errors and returns generic "internal error"
func NewErrorInterceptor(logger *slog.Logger) connect.Interceptor {
	return &errorClassifier{logger: logger}
}

func (e *errorClassifier) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err == nil {
			return resp, nil
		}

		// Already a connect error — handler explicitly classified it.
		if _, ok := err.(*connect.Error); ok {
			return resp, err
		}

		// Unclassified error — log and return generic internal error.
		// Domain errors (NotFound, AlreadyExists, Validation) should be mapped
		// to connect codes by handlers via their mapError functions.
		e.logger.ErrorContext(ctx, "unhandled error",
			"procedure", req.Spec().Procedure,
			"error", err,
		)
		return resp, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
}

func (e *errorClassifier) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (e *errorClassifier) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
