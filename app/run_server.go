package app

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func serverDefaultOptions() *RunAppOptions {
	return &RunAppOptions{
		RunPublicHTTPServer:   true,
		RunInternalHTTPServer: true,
	}
}

func RunServer(optionsList ...RunOption) error {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	defaults := serverDefaultOptions()
	defaults.fxOptions = []fx.Option{
		fx.WithLogger(func() fxevent.Logger {
			return FxLogger{logger: bootLogger}
		}),
	}
	options := optionutil.ApplyOptions(defaults, optionsList...)

	if len(options.DotEnvFiles) > 0 {
		if err := godotenv.Load(options.DotEnvFiles...); err != nil {
			return fmt.Errorf("loading dotenv files: %w", err)
		}
	}

	applyEnvDefaults(bootLogger)

	fx.New(
		NewFxAppOptions(options),
		fx.StopTimeout(shutdownTimeout),
	).Run()

	return nil
}
