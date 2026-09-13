// Package tui provides small Bubble Tea building blocks (a multi-field
// form, and a spinner-then-result task runner) shared by the CLI's
// account commands.
package tui

import "github.com/charmbracelet/lipgloss"

// colorAccent and colorMuted are the CommitIn wordmark's colors: off-white
// (#f6f4f0) and black, sampled from the logo. colorGood/colorBad stay green
// and red -- those are universal success/failure conventions, not branding.
var (
	colorAccent = lipgloss.Color("#f6f4f0")
	colorMuted  = lipgloss.Color("241")
	colorGood   = lipgloss.Color("42")
	colorBad    = lipgloss.Color("203")

	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	labelStyle        = lipgloss.NewStyle().Foreground(colorMuted)
	focusedLabelStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	helpStyle         = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	successStyle      = lipgloss.NewStyle().Foreground(colorGood).Bold(true)
	errStyle          = lipgloss.NewStyle().Foreground(colorBad).Bold(true)
	spinnerStyle      = lipgloss.NewStyle().Foreground(colorAccent)
)

func boxStyle(border lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1)
}
