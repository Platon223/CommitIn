package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/apiclient"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/prompt"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var email, password string

	c := &cobra.Command{
		Use:   "login",
		Short: "Log in to your CommitIn account",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			url := resolveAPIURL(cfg)

			in := prompt.NewReader(out)
			if email == "" {
				if email, err = in.Line("Email: "); err != nil {
					return err
				}
			}
			if password == "" {
				if password, err = in.Hidden("Password: "); err != nil {
					return err
				}
			}

			resp, err := apiclient.New(url).Login(cmd.Context(), email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			cfg.Token = resp.Token
			cfg.APIURL = url
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save session: %w", err)
			}

			path, _ := config.Path()
			fmt.Fprintf(out, "✓ logged in as %s <%s>\n", resp.User.Username, resp.User.Email)
			fmt.Fprintf(out, "  session saved to %s\n", path)
			return nil
		},
	}

	c.Flags().StringVar(&email, "email", "", "account email (prompted if omitted)")
	c.Flags().StringVar(&password, "password", "", "account password (prompted if omitted; avoid on shared machines)")
	return c
}
