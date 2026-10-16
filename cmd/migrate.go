package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate",
		Aliases: []string{"m"},
		Short:   "Database migration commands",
	}

	cmd.AddCommand(
		newMigrateUpCommand(),
		newMigrateDownCommand(),
		newMigrateStatusCommand(),
		newMigrateCreateCommand(),
	)

	return cmd
}

func newMigrateUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Aliases: []string{"u"},
		Short:   "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			module, err := cmd.Flags().GetString("module")
			if err != nil {
				return err
			}

			return app.RunMigrationUp(app.WithRunModule(module), app.WithDotEnvFiles(dotEnvs...))
		},
	}
	cmd.Flags().String("module", "", "module name")

	return cmd
}

func newMigrateDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "down",
		Aliases: []string{"d"},
		Short:   "Roll back the latest migration of one module (--module) or of every module (--all)",
		Example: "synclet migrate down --module pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			module, err := cmd.Flags().GetString("module")
			if err != nil {
				return err
			}

			all, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}

			confirmed, err := cmd.Flags().GetBool("yes")
			if err != nil {
				return err
			}

			return app.RunMigrationDown(app.MigrationDownTarget{
				Module:    module,
				All:       all,
				Confirmed: confirmed,
			}, app.WithDotEnvFiles(dotEnvs...))
		},
	}
	cmd.Flags().String("module", "", "module to roll back")
	cmd.Flags().Bool("all", false, "roll back the latest migration of every module")
	cmd.Flags().Bool("yes", false, "confirm the rollback when ENVIRONMENT is not development")

	return cmd
}

func newMigrateStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"s"},
		Short:   "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			module, err := cmd.Flags().GetString("module")
			if err != nil {
				return err
			}

			return app.RunMigrationStatus(app.WithRunModule(module), app.WithDotEnvFiles(dotEnvs...))
		},
	}
	cmd.Flags().String("module", "", "module name")

	return cmd
}

func newMigrateCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "create <module> <name>",
		Short:   "Create a new migration file",
		Example: "synclet migrate create auth add_api_tokens",
		Aliases: []string{"c"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return cmd.Help()
			}

			return app.RunMigrationsCreate(args[0], strings.Join(args[1:], " "))
		},
	}
}
