package app

import (
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

func RunServer(optionsList ...RunOption) {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	defaults := serverDefaultOptions()
	defaults.fxOptions = []fx.Option{
		fx.WithLogger(func() fxevent.Logger {
			return FxLogger{logger: bootLogger}
		}),
	}
	options := optionutil.ApplyOptions(defaults, optionsList...)

	for _, file := range options.DotEnvFiles {
		if err := godotenv.Load(file); err != nil {
			bootLogger.Error("can't load dotenv file", slog.Any("file", file), slog.Any("error", err))
		}
	}

	applyEnvDefaults(bootLogger)

	fx.New(
		NewFxAppOptions(options),
		fx.StopTimeout(shutdownTimeout),
	).Run()
}
