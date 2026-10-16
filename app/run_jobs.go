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

func RunJobs(optionsList ...RunOption) error {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	options := optionutil.ApplyOptions(&RunAppOptions{
		RunJobs: true,
		fxOptions: []fx.Option{
			fx.WithLogger(func() fxevent.Logger {
				return FxLogger{logger: bootLogger}
			}),
		},
	}, optionsList...)

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
