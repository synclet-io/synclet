package app

import (
	"context"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestDockerStartupCleanup_RunsOutsideStartupDeadline(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})

	cleanup := startupCleanupFunc(func(ctx context.Context) error {
		defer close(finished)

		close(started)

		<-ctx.Done()

		return ctx.Err()
	})

	app := fx.New(
		fx.NopLogger,
		fx.Supply(cleanup),
		fx.Provide(func() *logging.Logger { return nil }),
		dockerStartupCleanupModule(),
	)
	require.NoError(t, app.Err())

	startCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	require.NoError(t, app.Start(startCtx))
	require.NoError(t, startCtx.Err(), "startup waited for the cleanup until the startup deadline")

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("startup cleanup did not run")
	}

	select {
	case <-finished:
		t.Fatal("cleanup ended together with startup")
	case <-time.After(300 * time.Millisecond):
	}

	require.NoError(t, app.Stop(context.Background()))

	select {
	case <-finished:
	default:
		t.Fatal("stop returned before the cleanup ended")
	}
}
