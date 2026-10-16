package coordinator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexeck8s"
)

const internalSecretHeader = "X-Internal-Secret" //nolint:gosec // header name, not a credential

// authInterceptor attaches the internal API token to outgoing requests.
type authInterceptor struct {
	token string
}

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set(internalSecretHeader, i.token)

		return next(ctx, req)
	}
}

func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(internalSecretHeader, i.token)

		return conn
	}
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// clientOptions returns the Connect client options that authenticate against
// the internal ExecutorService. A missing secrets dir or token file yields no
// options, so the server's own configuration decides whether calls succeed.
func clientOptions(secretsDir string) ([]connect.ClientOption, error) {
	token, err := readAPIToken(secretsDir)
	if err != nil {
		return nil, err
	}

	if token == "" {
		return nil, nil
	}

	return []connect.ClientOption{connect.WithInterceptors(&authInterceptor{token: token})}, nil
}

func readAPIToken(secretsDir string) (string, error) {
	if secretsDir == "" {
		return "", nil
	}

	data, err := os.ReadFile(filepath.Join(secretsDir, pipelineexeck8s.APITokenSecretKey)) //nolint:gosec // path is built from a fixed key under the mounted secrets dir
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("reading internal API token: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
