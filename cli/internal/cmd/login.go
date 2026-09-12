package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/apiclient"
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
			in := prompt.NewReader(out)

			var err error
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

			client := apiclient.New(apiURL)
			resp, err := client.Login(cmd.Context(), email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			fmt.Fprintf(out, "✓ logged in as %s <%s>\n", resp.User.Username, resp.User.Email)
			fmt.Fprintf(out, "  token: %s\n", resp.Token)
			fmt.Fprintln(out, "  (not saved yet — local token storage lands in Week 1 Day 5)")
			return nil
		},
	}

	c.Flags().StringVar(&email, "email", "", "account email (prompted if omitted)")
	c.Flags().StringVar(&password, "password", "", "account password (prompted if omitted; avoid on shared machines)")
	return c
}
