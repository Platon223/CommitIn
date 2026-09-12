package cmd

import (
	"errors"
	"fmt"

	"github.com/Platon223/commitin/cli/internal/apiclient"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newSignupCmd() *cobra.Command {
	var email, username, password string

	c := &cobra.Command{
		Use:   "signup",
		Short: "Create a CommitIn account",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			url := resolveAPIURL(cfg)

			// Build a form only for whichever fields weren't given as flags,
			// remembering each one's slot so the values can be matched back up.
			var fields []tui.FormField
			emailIdx, usernameIdx, passwordIdx := -1, -1, -1
			if email == "" {
				emailIdx = len(fields)
				fields = append(fields, tui.FormField{Label: "Email", Placeholder: "you@example.com"})
			}
			if username == "" {
				usernameIdx = len(fields)
				fields = append(fields, tui.FormField{Label: "Username", Placeholder: "yourhandle"})
			}
			if password == "" {
				passwordIdx = len(fields)
				fields = append(fields, tui.FormField{Label: "Password", Password: true})
			}
			if len(fields) > 0 {
				values, err := tui.RunForm("Create a CommitIn account", fields)
				if err != nil {
					if errors.Is(err, tui.ErrCancelled) {
						fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
						return fmt.Errorf("signup cancelled")
					}
					return err
				}
				if emailIdx >= 0 {
					email = values[emailIdx]
				}
				if usernameIdx >= 0 {
					username = values[usernameIdx]
				}
				if passwordIdx >= 0 {
					password = values[passwordIdx]
				}
			}

			return tui.RunTask("Creating account...", func() ([]string, error) {
				resp, err := apiclient.New(url).Signup(cmd.Context(), email, username, password)
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
					fmt.Sprintf("Account created: %s <%s>", resp.User.Username, resp.User.Email),
					fmt.Sprintf("Session saved to %s", path),
				}, nil
			})
		},
	}

	c.Flags().StringVar(&email, "email", "", "account email (prompted if omitted)")
	c.Flags().StringVar(&username, "username", "", "account username (prompted if omitted)")
	c.Flags().StringVar(&password, "password", "", "account password (prompted if omitted; avoid on shared machines)")
	return c
}
