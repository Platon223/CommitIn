# Contributing to CommitIn

Thanks for helping out. This is a small, friendly project; issues and pull requests are welcome.

## Layout

```
backend/   Go HTTP API (accounts, sessions, scores, leaderboard, stats) on MongoDB
cli/       the cmtin binary (git hook logic, Claude client, terminal UI)
```

Two Go modules, joined by the `go.work` at the repo root. Requires Go 1.26+.

## Build and test

Run these in **each** module (`backend/` and `cli/`) before opening a PR:

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

Build the CLI with `cd cli && go build -o ../bin/cmtin ./cmd/cmtin`, and run the backend with `cd backend && cp .env.example .env && go run ./cmd/server` (you need a `MONGODB_URI` for any MongoDB: the free Atlas tier or a local `mongod` both work).

## Developing without spending API credits

Judging calls the Claude API, but you never need a real key to work on CommitIn. The Anthropic SDK honors `ANTHROPIC_BASE_URL`, so point it at a tiny local server that answers `POST /v1/messages` with a canned `tool_use` block for the `submit_verdict` tool. `cli/internal/claude/claude_test.go` shows the exact response shape (`score`, `roast`, `suggestion`), and a good, a bad, a malformed and a 500 response cover almost every path.

```bash
ANTHROPIC_API_KEY=fake ANTHROPIC_BASE_URL=http://127.0.0.1:PORT git commit -m "wip"
```

## Things worth knowing

- **The hook must never block a commit because of infrastructure.** Only a genuine "this message is bad" verdict may reject one. A missing key, a network error, a timeout, a malformed response, a merge commit: all of these must exit 0. If you touch `cli/internal/cmd/hook.go`, keep every early return that way.
- **Two constants must match by hand.** `claude.PassingScore` (cli) and `score.PassingScore` (backend) are the pass mark; they live in separate modules, so change both together.
- **The store code needs a real MongoDB.** `user`, `session` and `score` have no unit tests for that reason; verify changes against your own MongoDB and describe what you ran in the PR. Pure logic (validation, rendering, parsing) should have tests.
- **Bubble Tea needs a real TTY.** Signup, login, `init`, the spinner and the leaderboard won't start without one. To script them, spawn the process on a pty **with a controlling terminal** (`setsid` + `TIOCSCTTY`), otherwise it fails with "could not open a new TTY". Lip Gloss strips colors when output isn't a terminal, which is why tests assert on plain text; set `CLICOLOR_FORCE=1` to see them.
- **Scores are keyed by a random `attempt_id`, not a commit hash.** The commit-msg hook runs before the commit exists, and rejected attempts are scored too.
- **Don't commit secrets.** `.env` is git-ignored; keep it that way.

## Pull requests

- Keep them focused, and explain the behavior you changed and how you checked it.
- Add or update tests for logic you touch.
- Since this project judges commit messages: please write good ones. Imperative mood, say *what* and *why*.

By contributing you agree that your work is released under the [MIT License](LICENSE).
