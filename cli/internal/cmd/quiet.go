package cmd

import (
	"fmt"

	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

func newQuietCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quiet [on|off]",
		Short: "Pause or resume commit-msg judging, without uninstalling the hook",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}

			switch {
			case len(args) == 0:
				cfg.Quiet = !cfg.Quiet
			case args[0] == "on":
				cfg.Quiet = true
			case args[0] == "off":
				cfg.Quiet = false
			default:
				return fmt.Errorf("usage: cmtin quiet [on|off]")
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			if cfg.Quiet {
				tui.PrintInfo(out, "quiet mode on -- commits won't be judged until you run `cmtin quiet off`")
			} else {
				tui.PrintSuccess(out, "quiet mode off -- commits will be judged again")
			}
			return nil
		},
	}
}
