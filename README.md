# CommitIn 🚧 Coming Soon. **Lock in** on commit messages with **CommitIn**.

<img width="170" height="170" alt="Cmtin" src="https://github.com/user-attachments/assets/85ba844b-c9ff-4778-b6d0-ea5bddad6685" />

**A CLI that judges your commit messages so your teammates don't have to.**

> ⭐ **This is just an idea right now, nothing is built.** If this repo hits **20 stars**, I'll start building it for real. Star it if you'd use this.

---

### 👋 Coming from LinkedIn?

Hey, thanks for checking this out. CommitIn started as a reaction to seeing way too many "fix stuff" and "asdf" commits out in the wild. Nothing's built yet. I want to see if this is actually worth building before I sink time into it. Star the repo if you'd use this, and if you've got a commit message horror story, drop it in the issues, it might end up as an example roast.

---

## What is CommitIn?

CommitIn hooks into `git commit` and instantly roasts lazy commit messages while giving real ones proper credit. No dashboard, no config files, no leaving your terminal, just an instant, honest (and slightly savage) verdict every time you commit.

The goal isn't just a laugh, it's to actually **lock in better commit habits** over time. Roast today, track your improvement tomorrow.

## Planned features

**Free**
- Instant commit message roasts, right in your terminal, on every commit
- A public leaderboard — invite friends, compete on commit quality
- Sign up / log in, entirely from the terminal — no browser required

**Pro**
- Metrics over time — track your commit quality trend and see if you're actually improving
- Upgrade with one command (`cmtin upgrade`), payment handled securely via Stripe

## Beyond the roast

CommitIn doesn't just judge you, it can also generate a *good* commit message for you, based on your actual diff. If you're staring at your terminal with no idea how to summarize what you just changed, CommitIn reads the diff and suggests a clear, properly formatted message for you to use or tweak.

## How it'll work

```bash
cmtin init
```

Installs a lightweight git hook in your repo. From then on, every `git commit` gets scored automatically:

```bash
git commit -m "fix stuff"
```

```
💀 "fix stuff" — groundbreaking. truly the pinnacle of documentation.
   Score: 2/10
```

No extra commands, no extra steps, it just runs.

## Architecture

<!-- architecture diagram goes here -->

## Stack

Being built in Go, with [Bubble Tea](https://github.com/charmbracelet/bubbletea) powering the interactive parts (leaderboard, login, stats).

## Status

Nothing built yet, waiting to see if this is worth building. See the note at the top of this README.

## License

MIT
