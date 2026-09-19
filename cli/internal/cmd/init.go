package cmd

import (
	"errors"
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
	{Command: "cmtin quiet", Description: "Pause or resume commit-msg judging"},
	{Command: "cmtin leaderboard", Description: "Show the public leaderboard"},
}

func newInitCmd() *cobra.Command {
	var anthropicKey string

	c := &cobra.Command{
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

			changed := false
			if anthropicKey != "" {
				cfg.AnthropicKey = anthropicKey
				changed = true
			}
			if cfg.EffectiveAnthropicKey() == "" {
				values, err := tui.RunForm("Anthropic API key", []tui.FormField{
					{Label: "Anthropic API key (used to judge your commits)", Placeholder: "sk-ant-...", Password: true},
				})
				if err != nil {
					if errors.Is(err, tui.ErrCancelled) {
						fmt.Fprintln(out, "cancelled")
						return fmt.Errorf("init cancelled")
					}
					return err
				}
				cfg.AnthropicKey = values[0]
				changed = true
			}
			if changed {
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
			}
			if cfg.EffectiveAnthropicKey() == "" {
				tui.PrintError(out, "no Anthropic key set -- roasts will be skipped until you run `cmtin init` again or set ANTHROPIC_API_KEY")
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

	c.Flags().StringVar(&anthropicKey, "anthropic-key", "", "Anthropic API key (prompted if omitted and not already set)")
	return c
}
