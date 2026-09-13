package tui

import (
	_ "embed"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

//go:embed assets/logo.ans
var logoRaw string

// logoWidth is the embedded logo's rendered width, in terminal columns.
const logoWidth = 104

// Logo returns the CommitIn wordmark centered for a terminal termWidth
// columns wide, or "" if the terminal is too narrow to show it without
// wrapping.
func Logo(termWidth int) string {
	if termWidth < logoWidth {
		return ""
	}
	lines := strings.Split(strings.TrimRight(logoRaw, "\n"), "\n")
	for i, l := range lines {
		lines[i] = lipgloss.PlaceHorizontal(termWidth, lipgloss.Center, l)
	}
	return strings.Join(lines, "\n")
}
