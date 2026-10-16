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
	return runGooseCommand(nil, func(_ string, db *sql.DB) error {
		return goose.Up(db, ".")
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

	return runGooseCommand(confirm, func(module string, sqlDB *sql.DB) error {
		version, err := goose.GetDBVersion(sqlDB)
		if err != nil {
			return err
		}

		if version == 0 {
			fmt.Printf("module %s: nothing to roll back\n", module)

			return nil
		}

		fmt.Printf("module %s: rolling back version %d\n", module, version)

		return goose.Down(sqlDB, ".")
	}, opts...)
}

func RunMigrationStatus(opts ...RunOption) error {
	return runGooseCommand(nil, func(_ string, db *sql.DB) error {
		return goose.Status(db, ".")
	}, opts...)
}

func runGooseCommand(check func() error, proceed func(module string, db *sql.DB) error, optionsList ...RunOption) error {
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

		locker, err := lock.NewPostgresSessionLocker()
		if err != nil {
			return err
		}

		for _, migration := range migrations.Sources() {
			if !options.needToRunModule(migration.Module) {
				continue
			}

			goose.WithSessionLocker(locker)
			goose.SetTableName(migration.Module + "_goose_db_version")
			goose.SetBaseFS(migration.FS)

			if err := proceed(migration.Module, sqlDB); err != nil {
				return fmt.Errorf("module %s: %w", migration.Module, err)
			}
		}

		return nil
	}, NewFxAppOptions(options))
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
