package tui

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PeriodView is one window's numbers, as shown by the stats card.
type PeriodView struct {
	Attempts int
	Average  float64
	Rejected int
}

func (p PeriodView) rejectionRate() float64 {
	if p.Attempts == 0 {
		return 0
	}
	return float64(p.Rejected) / float64(p.Attempts) * 100
}

// StatsView is everything the stats card needs: the current window, the
// equal-length window before it, and the pass mark the gate uses.
type StatsView struct {
	WindowDays   int
	PassingScore int
	Current      PeriodView
	Previous     PeriodView
}

// deltaKind says how a change should be colored.
type deltaKind int

const (
	deltaNeutral deltaKind = iota // volume: more or fewer isn't good or bad
	deltaHigherIsBetter
	deltaLowerIsBetter
)

// formatDelta renders cur-prev as "▲ +1.67" / "▼ -0.40" / "no change",
// colored green when the change is an improvement and red when it's a
// regression (muted for neutral metrics). unit is appended to the number.
func formatDelta(cur, prev float64, decimals int, unit string, kind deltaKind) string {
	d := cur - prev
	if math.Abs(d) < math.Pow(10, -float64(decimals))/2 {
		return labelStyle.Render("no change")
	}
	arrow := "▲"
	if d < 0 {
		arrow = "▼"
	}
	text := fmt.Sprintf("%s %+.*f%s", arrow, decimals, d, unit)

	improved := (d > 0) == (kind == deltaHigherIsBetter)
	switch {
	case kind == deltaNeutral:
		return labelStyle.Render(text)
	case improved:
		return successStyle.Render(text)
	default:
		return errStyle.Render(text)
	}
}

const (
	statsLabelW = 16
	statsValueW = 14
)

func statsRow(label, value, delta string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Width(statsLabelW).Render(label) +
		lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Width(statsValueW).Render(value) +
		delta
}

// renderStats builds the stats card. Deltas only appear when both windows
// have data -- comparing against an empty window would be meaningless.
func renderStats(v StatsView) string {
	cur, prev := v.Current, v.Previous
	comparable := cur.Attempts > 0 && prev.Attempts > 0
	none := labelStyle.Render("no earlier data to compare")

	delta := func(s string) string {
		if !comparable {
			return none
		}
		return s
	}

	lines := []string{
		titleStyle.Render("Your CommitIn stats"),
		labelStyle.Render(fmt.Sprintf("last %d days, compared with the %d before", v.WindowDays, v.WindowDays)),
		"",
		statsRow("Average score", fmt.Sprintf("%.2f / 10", cur.Average),
			delta(formatDelta(cur.Average, prev.Average, 2, "", deltaHigherIsBetter))),
		statsRow("Judged commits", fmt.Sprintf("%d", cur.Attempts),
			delta(formatDelta(float64(cur.Attempts), float64(prev.Attempts), 0, "", deltaNeutral))),
		statsRow("Rejected", fmt.Sprintf("%d (%.0f%%)", cur.Rejected, cur.rejectionRate()),
			delta(formatDelta(cur.rejectionRate(), prev.rejectionRate(), 1, " pts", deltaLowerIsBetter))),
		"",
		helpStyle.Render(fmt.Sprintf("rejected = scored below %d, blocked by the commit gate", v.PassingScore)),
	}
	return boxStyle(colorMuted).Render(strings.Join(lines, "\n"))
}

// PrintStats writes the stats card.
func PrintStats(w io.Writer, v StatsView) {
	fmt.Fprintln(w, renderStats(v))
}

// PrintEmptyStats is the "nothing to show yet" state for a brand-new
// account, or one with no judged commits in the window.
func PrintEmptyStats(w io.Writer, windowDays int) {
	body := titleStyle.Render("No stats yet") + "\n" +
		labelStyle.Render(fmt.Sprintf("Nothing judged in the last %d days.", windowDays)) + "\n" +
		labelStyle.Render("Run `cmtin init` in a repo, then commit -- your history builds up here.")
	fmt.Fprintln(w, boxStyle(colorMuted).Render(body))
}
