package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Platon223/commitin/cli/internal/claude"
	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/gitcli"
	"github.com/Platon223/commitin/cli/internal/tui"
	"github.com/spf13/cobra"
)

// judgeTimeout bounds the Claude call so a slow network can't hang a commit
// indefinitely; a timeout is just another infra failure and fails open.
// 30s gives normal latency variance and the SDK's own retry-on-transient-error
// behavior room to actually succeed instead of tripping the deadline.
const judgeTimeout = 30 * time.Second

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
		// Every early-return path below is deliberate: anything that isn't a
		// genuine "the AI judged this message bad" result must exit 0 (nil
		// error) so the commit is never blocked by a merge, an empty diff, a
		// missing key, or a Claude API/network failure. Only the final
		// bad-verdict branch returns a non-nil error.
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if gitcli.IsMerging() {
				// A merge's diff --cached (index vs. one parent) doesn't
				// represent "this commit's changes", and merge messages are
				// auto-generated -- nothing useful to judge here.
				fmt.Fprintln(out, "CommitIn: skipping merge commit")
				return nil
			}

			cfg, err := config.Load()
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("CommitIn: could not read local config, skipping: %v", err))
				return nil
			}
			key := cfg.EffectiveAnthropicKey()
			if key == "" {
				fmt.Fprintln(out, "CommitIn: no Anthropic API key set, skipping (run `cmtin init` to add one)")
				return nil
			}

			msg, err := os.ReadFile(args[0])
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("CommitIn: could not read commit message, skipping: %v", err))
				return nil
			}

			diff, err := gitcli.DiffCached()
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("CommitIn: could not read staged diff, skipping: %v", err))
				return nil
			}
			if diff == "" {
				fmt.Fprintln(out, "CommitIn: nothing staged, skipping")
				return nil
			}
			diff, _ = gitcli.TruncateDiff(diff, gitcli.DefaultMaxDiffLines)

			ctx, cancel := context.WithTimeout(cmd.Context(), judgeTimeout)
			defer cancel()

			var verdict *claude.Verdict
			tui.RunSpinner("Judging your commit message...", func() {
				verdict, err = claude.New(key, "").Judge(ctx, string(msg), diff)
			})
			if err != nil {
				// Infra failure (network, auth, rate limit, malformed
				// response) -- never block a commit over this.
				tui.PrintError(out, fmt.Sprintf("CommitIn: judge request failed, skipping: %v", err))
				return nil
			}

			if verdict.Good() {
				tui.PrintSuccess(out, fmt.Sprintf("%d/10 -- %s", verdict.Score, verdict.Roast))
				return nil
			}

			tui.PrintError(out, fmt.Sprintf("%d/10 -- %s", verdict.Score, verdict.Roast))
			fmt.Fprintf(out, "\nTry this instead:\n\n  %s\n\n", verdict.Suggestion)
			return fmt.Errorf("commit message rejected")
		},
	}
}
