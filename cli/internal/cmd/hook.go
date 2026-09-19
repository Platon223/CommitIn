package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/Platon223/commitin/cli/internal/apiclient"
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

// submitScoreTimeout bounds the (much smaller/faster) score submission call.
const submitScoreTimeout = 10 * time.Second

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
		// error) so the commit is never blocked by quiet mode, a merge, a
		// missing key, an empty diff, or a Claude API/network failure. Only
		// the final bad-verdict branch returns a non-nil error.
		//
		// PrintInfo is for expected no-ops (nothing went wrong, there's just
		// nothing to judge); PrintError is reserved for things that actually
		// went wrong, even though they still fail open.
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			cfg, err := config.Load()
			if err != nil {
				tui.PrintError(out, fmt.Sprintf("CommitIn: could not read local config, skipping: %v", err))
				return nil
			}
			if cfg.Quiet {
				tui.PrintInfo(out, "quiet mode is on, skipping (run `cmtin quiet off` to re-enable)")
				return nil
			}

			if gitcli.IsMerging() {
				// A merge's diff --cached (index vs. one parent) doesn't
				// represent "this commit's changes", and merge messages are
				// auto-generated -- nothing useful to judge here.
				tui.PrintInfo(out, "skipping merge commit")
				return nil
			}

			key := cfg.EffectiveAnthropicKey()
			if key == "" {
				tui.PrintInfo(out, "no Anthropic API key set, skipping (run `cmtin init` to add one)")
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
				tui.PrintInfo(out, "nothing staged, skipping")
				return nil
			}
			diff, _ = gitcli.TruncateDiff(diff, gitcli.DefaultMaxDiffLines)

			judgeCtx, cancel := context.WithTimeout(cmd.Context(), judgeTimeout)
			defer cancel()

			var verdict *claude.Verdict
			tui.RunSpinner("Judging your commit message...", func() {
				verdict, err = claude.New(key, "").Judge(judgeCtx, string(msg), diff)
			})
			if err != nil {
				// Infra failure (network, auth, rate limit, malformed
				// response) -- never block a commit over this.
				tui.PrintError(out, fmt.Sprintf("CommitIn: judge request failed, skipping: %v", err))
				return nil
			}

			if verdict.Good() {
				tui.PrintSuccess(out, fmt.Sprintf("%d/10 -- %s", verdict.Score, verdict.Roast))
				submitScore(cmd, cfg, verdict.Score)
				return nil
			}

			tui.PrintError(out, fmt.Sprintf("%d/10 -- %s", verdict.Score, verdict.Roast))
			tui.PrintSuggestion(out, verdict.Suggestion)
			submitScore(cmd, cfg, verdict.Score)
			return fmt.Errorf("commit message rejected")
		},
	}
}

// submitScore records a judged attempt (accepted or rejected) for the
// leaderboard/stats. It's silent and best-effort: skipped entirely if the
// user isn't logged in (score submission needs the CommitIn session token,
// separately from the Anthropic key that judging itself needs -- so
// `cmtin logout` naturally stops contributing to the leaderboard even
// though local judging keeps working), and any failure is swallowed rather
// than printed, since it's background bookkeeping, not the roast itself.
func submitScore(cmd *cobra.Command, cfg *config.Config, score int) {
	if cfg.Token == "" {
		return
	}
	repoName, err := gitcli.RepoName()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), submitScoreTimeout)
	defer cancel()
	_ = apiclient.New(resolveAPIURL(cfg)).SubmitScore(ctx, cfg.Token, score, randomAttemptID(), repoName)
}

// randomAttemptID returns an opaque per-attempt identifier for score dedup.
func randomAttemptID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read practically never fails on supported platforms;
		// fall back rather than let this block a commit either way.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
