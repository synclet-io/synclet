package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newUsersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Operator commands for user accounts",
	}

	cmd.AddCommand(newUsersResetLinkCommand())

	return cmd
}

func newUsersResetLinkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-link",
		Short: "Print a single-use password reset link for a user",
		Long: `Creates a password reset link for the user with the given email and prints it to
stdout. The link expires in 1 hour and can be used once; issuing a new link
invalidates the previous one. Works with or without SMTP configured.`,
		Example:       "synclet users reset-link --email jane@example.com",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			email, err := cmd.Flags().GetString("email")
			if err != nil {
				return err
			}

			return app.RunUserResetLink(cmd.OutOrStdout(), email, app.WithDotEnvFiles(dotEnvs...))
		},
	}
	cmd.Flags().String("email", "", "email of the user")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}
