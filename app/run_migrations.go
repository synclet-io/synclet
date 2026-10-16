package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/go-pnp/go-pnp/fxutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox/txoutboxdbstate"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/pkg/migrations"
)

func init() {
	migrations.Register("tx_outbox", txoutboxdbstate.MigrationsFS)
}

func RunMigrationUp(opts ...RunOption) error {
	return runGooseCommand(nil, func(ctx context.Context, module string, provider *goose.Provider) error {
		results, err := provider.Up(ctx)
		for _, result := range results {
			fmt.Printf("module %s: applied %s (%s)\n", module, path.Base(result.Source.Path), result.Duration.Round(time.Millisecond))
		}

		return err
	}, opts...)
}

// MigrationDownTarget selects what `migrate down` rolls back: the latest
// migration of one module, or of every module when All is set.
type MigrationDownTarget struct {
	Module string
	All    bool
	// Confirmed must be set to roll back when ENVIRONMENT is anything other
	// than unset or "development".
	Confirmed bool
}

func RunMigrationDown(target MigrationDownTarget, opts ...RunOption) error {
	switch {
	case target.Module != "" && target.All:
		return errors.New("migrate down: --module and --all are mutually exclusive")
	case target.Module == "" && !target.All:
		return errors.New("migrate down: pass --module <name> to roll back one module, or --all to roll back every module")
	case target.Module != "":
		if err := validateMigrationModule(target.Module); err != nil {
			return err
		}

		opts = append(opts, WithRunModule(target.Module))
	}

	confirm := func() error {
		if env := os.Getenv("ENVIRONMENT"); env != "" && env != "development" && !target.Confirmed {
			return fmt.Errorf("migrate down: ENVIRONMENT is %q, pass --yes to confirm the rollback", env)
		}

		return nil
	}

	return runGooseCommand(confirm, func(ctx context.Context, module string, provider *goose.Provider) error {
		version, err := provider.GetDBVersion(ctx)
		if err != nil {
			return err
		}

		if version == 0 {
			fmt.Printf("module %s: nothing to roll back\n", module)

			return nil
		}

		fmt.Printf("module %s: rolling back version %d\n", module, version)

		_, err = provider.Down(ctx)

		return err
	}, opts...)
}

func RunMigrationStatus(opts ...RunOption) error {
	return runGooseCommand(nil, func(ctx context.Context, module string, provider *goose.Provider) error {
		statuses, err := provider.Status(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("module %s:\n", module)

		for _, status := range statuses {
			appliedAt := "Pending"
			if status.State == goose.StateApplied {
				appliedAt = status.AppliedAt.UTC().Format(time.DateTime)
			}

			fmt.Printf("  %-19s  %s\n", appliedAt, path.Base(status.Source.Path))
		}

		return nil
	}, opts...)
}

// runGooseCommand runs proceed for every selected module. Each module's
// provider holds a Postgres advisory session lock for the duration of each
// operation, so concurrent migrators (one per server replica) run one at a time.
func runGooseCommand(check func() error, proceed func(ctx context.Context, module string, provider *goose.Provider) error, optionsList ...RunOption) error {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	options := optionutil.ApplyOptions(&RunAppOptions{
		fxOptions: []fx.Option{
			fx.WithLogger(func() fxevent.Logger {
				return FxLogger{logger: bootLogger}
			}),
		},
	}, optionsList...)

	if len(options.DotEnvFiles) > 0 {
		if err := godotenv.Load(options.DotEnvFiles...); err != nil {
			return err
		}
	}

	applyPlainEnvDefaults()

	if check != nil {
		if err := check(); err != nil {
			return err
		}
	}

	return fxutil.RunJob2(func(ctx context.Context, logger *logging.Logger, db *gorm.DB) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}

		for _, migration := range migrations.Sources() {
			if !options.needToRunModule(migration.Module) {
				continue
			}

			provider, err := newMigrationProvider(sqlDB, migration)
			if err != nil {
				return fmt.Errorf("module %s: %w", migration.Module, err)
			}

			if err := proceed(ctx, migration.Module, provider); err != nil {
				return fmt.Errorf("module %s: %w", migration.Module, err)
			}
		}

		return nil
	}, NewFxAppOptions(options))
}

func newMigrationProvider(sqlDB *sql.DB, migration migrations.Source) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, err
	}

	return goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		migration.FS,
		goose.WithTableName(migration.Module+"_goose_db_version"),
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	)
}

func RunMigrationsCreate(module, migrationName string) error {
	if err := validateMigrationModule(module); err != nil {
		return err
	}

	migrationName = strings.ToLower(migrationName)
	migrationName = regexp.MustCompile(`[^a-z0-9 _-]`).ReplaceAllString(migrationName, "")
	migrationName = strings.NewReplacer("-", "_", " ", "_").Replace(migrationName)

	now := time.Now().UTC().Format("20060102150405")
	migrationName = fmt.Sprintf("%s_%s", now, migrationName)

	migrationPath := path.Join("./modules/", module, module+"dbstate", migrationName+".sql")

	content := `-- +goose Up
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
`

	if err := os.WriteFile(migrationPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to create migration file '%s': %w", migrationPath, err)
	}

	fmt.Printf("Created migration: %s\n", migrationPath)

	return nil
}

func validateMigrationModule(module string) error {
	names := make([]string, 0, len(migrations.Sources()))
	for _, ms := range migrations.Sources() {
		if ms.Module == module {
			return nil
		}

		names = append(names, ms.Module)
	}

	return fmt.Errorf("invalid module '%s' (valid: %s)", module, strings.Join(names, ", "))
}
