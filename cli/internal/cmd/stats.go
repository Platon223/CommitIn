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

const statsTimeout = 15 * time.Second

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show your own score history and trend",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load local config: %w", err)
			}
			// Stats are per-user, so unlike the public leaderboard this needs
			// a saved session.
			if err := requireLogin(out, cfg); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), statsTimeout)
			defer cancel()

			st, err := apiclient.New(resolveAPIURL(cfg)).Stats(ctx, cfg.Token)
			if err != nil {
				if apiclient.IsUnauthorized(err) {
					tui.PrintError(out, "your session has expired or was revoked -- run `cmtin login` again")
				} else {
					tui.PrintError(out, fmt.Sprintf("could not load stats: %v", err))
				}
				return err
			}

			if st.Current.Attempts == 0 {
				tui.PrintEmptyStats(out, st.WindowDays)
				return nil
			}
			tui.PrintStats(out, tui.StatsView{
				WindowDays:   st.WindowDays,
				PassingScore: st.PassingScore,
				Current:      tui.PeriodView{Attempts: st.Current.Attempts, Average: st.Current.AverageScore, Rejected: st.Current.Rejected},
				Previous:     tui.PeriodView{Attempts: st.Previous.Attempts, Average: st.Previous.AverageScore, Rejected: st.Previous.Rejected},
			})
			return nil
		},
	}
}
