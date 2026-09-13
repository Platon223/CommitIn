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
