package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/githook"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

var initCommandRows = []tui.CommandRow{
	{Command: "cmtin signup", Description: "Create a CommitIn account"},
	{Command: "cmtin login", Description: "Log in to your CommitIn account"},
	{Command: "cmtin logout", Description: "Log out and clear the local session"},
	{Command: "cmtin init", Description: "Install the commit-msg hook in this repo"},
	{Command: "cmtin uninstall", Description: "Remove the hook"},
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up CommitIn in this git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			if err := requireLogin(out, cfg); err != nil {
				return err
			}

			gitDir, err := githook.GitDir()
			if err != nil {
				tui.PrintError(out, err.Error())
				return err
			}
			if err := githook.Install(gitDir); err != nil {
				tui.PrintError(out, err.Error())
				return err
			}

			fmt.Fprint(out, tui.InitBanner(tui.TermWidth(120), initCommandRows))
			return nil
		},
	}
}
