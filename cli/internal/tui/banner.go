package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// InitBanner renders the success view for `cmtin init`: the centered logo
// (skipped if the terminal is too narrow to show it without wrapping), a
// table of available commands, and a reminder of how to uninstall.
func InitBanner(termWidth int, rows []CommandRow) string {
	var b strings.Builder
	if logo := Logo(termWidth); logo != "" {
		b.WriteString(logo)
		b.WriteString("\n\n")
	}
	b.WriteString(center(termWidth, CommandTable(rows)))
	b.WriteString("\n\n")
	b.WriteString(center(termWidth, successStyle.Render("Uninstall anytime with: cmtin uninstall")))
	b.WriteString("\n")
	return b.String()
}

func center(width int, s string) string {
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, s)
}
