package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// CommandRow is one row of a CommandTable.
type CommandRow struct {
	Command     string
	Description string
}

// CommandTable renders a bordered command reference table in the CommitIn
// brand style (cream/black, no accent color beyond the header).
func CommandTable(rows []CommandRow) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("COMMAND", "DESCRIPTION").
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Padding(0, 1)
			case col == 0:
				return lipgloss.NewStyle().Bold(true).Padding(0, 1)
			default:
				return lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)
			}
		})
	for _, r := range rows {
		t.Row(r.Command, r.Description)
	}
	return t.Render()
}

// TermWidth returns the current terminal's column width, or fallback if it
// can't be determined (not a terminal, redirected output, etc).
func TermWidth(fallback int) int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return fallback
}
