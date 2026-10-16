package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/go-pnp/go-pnp/fxutil"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// RunUserResetLink issues a password reset link for the user with the given
// email and writes the link, alone on one line, to out. The per-user reset
// cooldown does not apply.
func RunUserResetLink(out io.Writer, email string, optionsList ...RunOption) error {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	options := optionutil.ApplyOptions(&RunAppOptions{
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

	applyPlainEnvDefaults()

	return fxutil.RunJob1(func(ctx context.Context, issue *authservice.IssuePasswordResetLink) error {
		link, err := issue.Execute(ctx, authservice.IssuePasswordResetLinkParams{Email: email})
		if err != nil {
			if _, ok := lo.ErrorsAs[authservice.NotFoundError](err); ok {
				return errors.New("no user with email " + email)
			}

			return err
		}

		_, err = fmt.Fprintln(out, link.URL)

		return err
	}, NewFxAppOptions(options))
}
