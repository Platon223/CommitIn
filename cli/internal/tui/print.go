package tui

import (
	"fmt"
	"io"
)

// PrintError renders a one-line styled error message. Unlike RunForm/RunTask,
// this doesn't launch an interactive program, so it works without a TTY --
// use it for synchronous checks (e.g. "not logged in") that have nothing to
// spin on.
func PrintError(w io.Writer, msg string) {
	fmt.Fprintln(w, errStyle.Render("✗ "+msg))
}

// PrintSuccess renders a one-line styled success message. See PrintError.
func PrintSuccess(w io.Writer, msg string) {
	fmt.Fprintln(w, successStyle.Render("✓ "+msg))
}

// PrintInfo renders a one-line neutral, muted message -- for expected
// no-ops (skipping a merge commit, nothing staged, quiet mode), as opposed
// to PrintError, which should read as something actually went wrong.
func PrintInfo(w io.Writer, msg string) {
	fmt.Fprintln(w, labelStyle.Render("· "+msg))
}

// PrintSuggestion renders a "try this instead" block under a rejected
// verdict.
func PrintSuggestion(w io.Writer, suggestion string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, focusedLabelStyle.Render("Try this instead:"))
	fmt.Fprintln(w, "  "+suggestion)
	fmt.Fprintln(w)
}
