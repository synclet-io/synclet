package app

import (
	"context"
	"database/sql"
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
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/pkg/migrations"
)

func RunMigrationUp(opts ...RunOption) error {
	return runGooseCommand("up", func(db *sql.DB) error {
		return goose.Up(db, ".")
	}, opts...)
}

func RunMigrationDown(opts ...RunOption) error {
	return runGooseCommand("down", func(db *sql.DB) error {
		if err := goose.Down(db, "."); err != nil {
			if err.Error() == "no migration 0" {
				return nil
			}

			return err
		}

		return nil
	}, opts...)
}

func RunMigrationStatus(opts ...RunOption) error {
	return runGooseCommand("status", func(db *sql.DB) error {
		return goose.Status(db, ".")
	}, opts...)
}

func runGooseCommand(_ string, proceed func(db *sql.DB) error, optionsList ...RunOption) error {
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

			if err := proceed(sqlDB); err != nil {
				return fmt.Errorf("module %s: %w", migration.Module, err)
			}
		}

		return nil
	}, NewFxAppOptions(options))
}

func RunMigrationsCreate(module, migrationName string) error {
	found := false
	for _, ms := range migrations.Sources() {
		if ms.Module == module {
			found = true
			break
		}
	}

	if !found {
		names := make([]string, 0, len(migrations.Sources()))
		for _, ms := range migrations.Sources() {
			names = append(names, ms.Module)
		}

		return fmt.Errorf("invalid module '%s' (valid: %s)", module, strings.Join(names, ", "))
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
