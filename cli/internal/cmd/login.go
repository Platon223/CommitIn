package cmd

import (
	"errors"
	"fmt"

	"github.com/Platon223/commitin/cli/internal/apiclient"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var email, password string

	c := &cobra.Command{
		Use:   "login",
		Short: "Log in to your CommitIn account",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			url := resolveAPIURL(cfg)

			var fields []tui.FormField
			emailIdx, passwordIdx := -1, -1
			if email == "" {
				emailIdx = len(fields)
				fields = append(fields, tui.FormField{Label: "Email", Placeholder: "you@example.com"})
			}
			if password == "" {
				passwordIdx = len(fields)
				fields = append(fields, tui.FormField{Label: "Password", Password: true})
			}
			if len(fields) > 0 {
				values, err := tui.RunForm("Log in to CommitIn", fields)
				if err != nil {
					if errors.Is(err, tui.ErrCancelled) {
						fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
						return fmt.Errorf("login cancelled")
					}
					return err
				}
				if emailIdx >= 0 {
					email = values[emailIdx]
				}
				if passwordIdx >= 0 {
					password = values[passwordIdx]
				}
			}

			return tui.RunTask("Logging in...", func() ([]string, error) {
				resp, err := apiclient.New(url).Login(cmd.Context(), email, password)
				if err != nil {
					return nil, err
				}

				cfg.Token = resp.Token
				cfg.APIURL = url
				if err := config.Save(cfg); err != nil {
					return nil, fmt.Errorf("save session: %w", err)
				}

				path, _ := config.Path()
				return []string{
					fmt.Sprintf("Logged in as %s <%s>", resp.User.Username, resp.User.Email),
					fmt.Sprintf("Session saved to %s", path),
				}, nil
			})
		},
	}

	c.Flags().StringVar(&email, "email", "", "account email (prompted if omitted)")
	c.Flags().StringVar(&password, "password", "", "account password (prompted if omitted; avoid on shared machines)")
	return c
}
