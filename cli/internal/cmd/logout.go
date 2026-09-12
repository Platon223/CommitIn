package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/apiclient"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear the locally saved session",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			if cfg.Token == "" {
				fmt.Fprintln(out, "not logged in")
				return nil
			}

			// Best-effort: revoke the session on the backend, but a local
			// logout should still succeed even if the backend is unreachable.
			if err := apiclient.New(resolveAPIURL(cfg)).Logout(cmd.Context(), cfg.Token); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not revoke the session on the backend: %v\n", err)
			}

			cfg.Token = ""
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("clear local session: %w", err)
			}

			fmt.Fprintln(out, "✓ logged out")
			return nil
		},
	}
}
