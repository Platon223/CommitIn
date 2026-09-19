# Security

CommitIn handles account passwords, session tokens, and your Anthropic API key, and its hook reads your staged diffs, so security reports are taken seriously.

## Reporting a vulnerability

Please **do not open a public issue** for a security problem. Use GitHub's private reporting instead: open the repository's **Security** tab and choose **Report a vulnerability**. Include what you found, how to reproduce it, and its impact.

You'll get an acknowledgement as soon as it's seen, and a fix or an explanation after that. Please give a reasonable window to ship a fix before disclosing publicly.

## Scope

In scope: the backend's authentication and session handling, anything that could leak a user's email, password, token or API key, the hook or CLI executing something unintended, and any path where a diff or commit message reaches somewhere other than Anthropic (with the user's own key).

Only the latest `main` is supported.
