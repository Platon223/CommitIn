package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/apiclient"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear the locally saved session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			if cfg.Token == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "not logged in")
				return nil
			}

			return tui.RunTask("Logging out...", func() ([]string, error) {
				var warning string
				// Best-effort: revoke the session on the backend, but a local
				// logout should still succeed even if the backend is unreachable.
				if err := apiclient.New(resolveAPIURL(cfg)).Logout(cmd.Context(), cfg.Token); err != nil {
					warning = fmt.Sprintf("(could not revoke on the backend: %v)", err)
				}

				cfg.Token = ""
				if err := config.Save(cfg); err != nil {
					return nil, fmt.Errorf("clear local session: %w", err)
				}

				lines := []string{"Logged out"}
				if warning != "" {
					lines = append(lines, warning)
				}
				return lines, nil
			})
		},
	}
}
