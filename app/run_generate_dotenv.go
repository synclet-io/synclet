package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/fxutil"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func RunGenerateDotEnvFile() error {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	return fxutil.RunJob1(func(ctx context.Context, info configutil.ConfigsInfoIn) error {
		_, err := os.Stat(".env")

		var existingEnvs map[string]string

		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			existingEnvs, err = godotenv.Read(".env")
			if err != nil {
				return fmt.Errorf("read existing .env: %w", err)
			}
		}

		dotEnvFile, err := os.OpenFile(".env", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer dotEnvFile.Close() //nolint:errcheck

		for _, configInfo := range info.ConfigsInfo {
			newFields := make([]string, 0, len(configInfo.Fields))

			for _, config := range configInfo.Fields {
				if _, ok := existingEnvs[config.Key]; ok {
					continue
				}

				newFields = append(newFields, fmt.Sprintf("%s=\"%s\"", config.Key, config.DefaultValue))
			}

			if len(newFields) > 0 {
				_, _ = dotEnvFile.WriteString("# " + configInfo.ConfigType.String() + "\n")
				_, _ = dotEnvFile.WriteString(strings.Join(newFields, "\n") + "\n\n")
			}
		}

		return nil
	}, NewFxAppOptions(&RunAppOptions{
		fxOptions: []fx.Option{
			fx.WithLogger(func() fxevent.Logger {
				return FxLogger{logger: bootLogger}
			}),
		},
	}))
}
