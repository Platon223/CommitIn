package cmd

import (
	"fmt"
	"os"

	"github.com/Platon223/commitin/cli/internal/gitcli"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

// newHookCmd groups the commands the installed git hook shells out to.
// Hidden from `cmtin --help` since users never run these directly.
func newHookCmd() *cobra.Command {
	h := &cobra.Command{
		Use:    "hook",
		Short:  "Internal: commands run by the installed git hook",
		Hidden: true,
	}
	h.AddCommand(newHookCommitMsgCmd())
	return h
}

func newHookCommitMsgCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commit-msg <message-file>",
		Short: "Run by the installed commit-msg hook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			msg, err := os.ReadFile(args[0])
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("could not read commit message: %v", err))
				return err
			}

			diff, err := gitcli.DiffCached()
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("could not read staged diff: %v", err))
				return err
			}

			// No AI yet -- Week 3 sends this to Claude instead of printing it.
			fmt.Fprintln(out, "--- commit message ---")
			fmt.Fprint(out, string(msg))
			fmt.Fprintln(out, "--- staged diff ---")
			if diff == "" {
				fmt.Fprintln(out, "(empty)")
			} else {
				fmt.Fprint(out, diff)
			}
			return nil
		},
	}
}
