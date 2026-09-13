package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/githook"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove CommitIn's git hook from this repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			gitDir, err := githook.GitDir()
			if err != nil {
				tui.PrintError(out, err.Error())
				return err
			}

			removed, err := githook.Uninstall(gitDir)
			if err != nil {
				tui.PrintError(out, err.Error())
				return err
			}
			if !removed {
				fmt.Fprintln(out, "no CommitIn hook installed")
				return nil
			}

			tui.PrintSuccess(out, "commit-msg hook removed")
			return nil
		},
	}
}
