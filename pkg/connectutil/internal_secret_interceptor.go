package connectutil

import (
	"context"
	"crypto/subtle"
	"fmt"

	"connectrpc.com/connect"
)

// InternalSecretInterceptor validates a shared secret on internal API calls.
type InternalSecretInterceptor struct {
	secret string
}

// NewInternalSecretInterceptor creates a new interceptor that requires a matching
// X-Internal-Secret header on all requests.
func NewInternalSecretInterceptor(secret string) *InternalSecretInterceptor {
	return &InternalSecretInterceptor{secret: secret}
}

func (i *InternalSecretInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := i.validateSecret(req.Header().Get("X-Internal-Secret")); err != nil {
			return nil, err
		}

		return next(ctx, req)
	}
}

func (i *InternalSecretInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *InternalSecretInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := i.validateSecret(conn.RequestHeader().Get("X-Internal-Secret")); err != nil {
			return err
		}

		return next(ctx, conn)
	}
}

func (i *InternalSecretInterceptor) validateSecret(provided string) error {
	if i.secret == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("internal API token not configured"))
	}

	if subtle.ConstantTimeCompare([]byte(provided), []byte(i.secret)) != 1 {
		return connect.NewError(connect.CodeUnauthenticated, nil)
	}

	return nil
}
