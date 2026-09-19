package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Platon223/commitin/cli/internal/apiclient"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

const leaderboardTimeout = 15 * time.Second

func newLeaderboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "leaderboard",
		Short: "Show the public CommitIn leaderboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// Viewing the leaderboard needs no login -- it's public. A saved
			// username, if any, is only used to highlight your own row.
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), leaderboardTimeout)
			defer cancel()

			entries, err := apiclient.New(resolveAPIURL(cfg)).Leaderboard(ctx)
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("could not load leaderboard: %v", err))
				return err
			}
			if len(entries) == 0 {
				tui.PrintInfo(out, "leaderboard is empty -- be the first to make the cut (10 judged commits in the last 30 days)")
				return nil
			}

			rows := make([]tui.LeaderboardRow, len(entries))
			for i, e := range entries {
				rows[i] = tui.LeaderboardRow{
					Rank:          i + 1,
					Username:      e.Username,
					AverageScore:  e.AverageScore,
					CommitCount:   e.CommitCount,
					IsCurrentUser: cfg.Token != "" && cfg.Username != "" && e.Username == cfg.Username,
				}
			}
			tui.RunLeaderboard(rows)
			return nil
		},
	}
}
