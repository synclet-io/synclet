package app

import (
	"log/slog"
	"os"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func RunServer(optionsList ...RunOption) {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	options := optionutil.ApplyOptions(&RunAppOptions{
		RunPublicHTTPServer:   true,
		RunInternalHTTPServer: true,
		fxOptions: []fx.Option{
			fx.WithLogger(func() fxevent.Logger {
				return FxLogger{logger: bootLogger}
			}),
		},
	}, optionsList...)

	for _, file := range options.DotEnvFiles {
		if err := godotenv.Load(file); err != nil {
			bootLogger.Error("can't load dotenv file", slog.Any("file", file), slog.Any("error", err))
		}
	}

	fx.New(
		NewFxAppOptions(options),
		fx.StopTimeout(60*time.Second),
	).Run()
}
