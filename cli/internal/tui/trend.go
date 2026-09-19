package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// DayView is one UTC day's aggregate, as returned by the backend. Days with
// no attempts are simply absent.
type DayView struct {
	Date     string // YYYY-MM-DD, UTC
	Attempts int
	Average  float64
}

// dayCell is one slot in the chart: a calendar day that may or may not have
// data. "No commits" is deliberately distinct from "scored zero".
type dayCell struct {
	Date     time.Time
	Attempts int
	Average  float64
}

func (c dayCell) hasData() bool { return c.Attempts > 0 }

const dateLayout = "2006-01-02"

// buildDaySeries lays daily out on a continuous calendar of windowDays+1
// days ending at today (the backend's window [now-30d, now] touches 31 UTC
// dates), oldest first. If the backend reports a date later than the local
// clock's today -- clock skew across midnight -- the series ends there
// instead, so that day isn't silently dropped.
func buildDaySeries(daily []DayView, today time.Time, windowDays int) []dayCell {
	end := today.UTC().Truncate(24 * time.Hour)
	byDate := make(map[string]DayView, len(daily))
	for _, d := range daily {
		byDate[d.Date] = d
		if t, err := time.Parse(dateLayout, d.Date); err == nil && t.After(end) {
			end = t
		}
	}

	cells := make([]dayCell, 0, windowDays+1)
	for i := windowDays; i >= 0; i-- {
		day := end.AddDate(0, 0, -i)
		c := dayCell{Date: day}
		if d, ok := byDate[day.Format(dateLayout)]; ok {
			c.Attempts, c.Average = d.Attempts, d.Average
		}
		cells = append(cells, c)
	}
	return cells
}

const sparkGlyphs = "▁▂▃▄▅▆▇█"

// sparkline draws one glyph per cell: height tracks the day's average score
// (0-10), red when that average is below the pass mark, and a dim dot when
// there were no commits at all.
func sparkline(cells []dayCell, passing int) string {
	glyphs := []rune(sparkGlyphs)
	var b strings.Builder
	for _, c := range cells {
		if !c.hasData() {
			b.WriteString(labelStyle.Render("·"))
			continue
		}
		idx := int(c.Average/10*float64(len(glyphs)-1) + 0.5)
		idx = max(0, min(len(glyphs)-1, idx))
		style := lipgloss.NewStyle().Foreground(colorAccent)
		if c.Average < float64(passing) {
			style = lipgloss.NewStyle().Foreground(colorBad)
		}
		b.WriteString(style.Render(string(glyphs[idx])))
	}
	return b.String()
}

// weekBucket is up to seven consecutive days, summarized.
type weekBucket struct {
	Start, End time.Time
	Attempts   int
	Average    float64 // attempt-weighted across the days in the bucket
}

// weeklyBuckets chunks the series into 7-day buckets counted back from the
// newest day, so the newest bucket is always a full week and any partial one
// is the oldest. Returned oldest first, matching the sparkline above it.
func weeklyBuckets(cells []dayCell) []weekBucket {
	var out []weekBucket
	for end := len(cells); end > 0; end -= 7 {
		start := max(0, end-7)
		wb := weekBucket{Start: cells[start].Date, End: cells[end-1].Date}
		var weighted float64
		for _, c := range cells[start:end] {
			wb.Attempts += c.Attempts
			weighted += c.Average * float64(c.Attempts)
		}
		if wb.Attempts > 0 {
			wb.Average = weighted / float64(wb.Attempts)
		}
		out = append([]weekBucket{wb}, out...)
	}
	return out
}

// rangeLabel renders "Sep 13–19", or "Aug 30–Sep 5" across a month boundary.
func rangeLabel(start, end time.Time) string {
	if start.Equal(end) {
		return start.Format("Jan 2")
	}
	if start.Month() == end.Month() {
		return fmt.Sprintf("%s–%d", start.Format("Jan 2"), end.Day())
	}
	return fmt.Sprintf("%s–%s", start.Format("Jan 2"), end.Format("Jan 2"))
}

// renderTrend is the chart block: sparkline with date axis, then weekly
// rows. Returns nil if there's nothing to draw.
func renderTrend(daily []DayView, today time.Time, windowDays, passing int) []string {
	if today.IsZero() || len(daily) == 0 {
		return nil
	}
	cells := buildDaySeries(daily, today, windowDays)
	first, last := cells[0].Date.Format("Jan 2"), cells[len(cells)-1].Date.Format("Jan 2")
	gap := max(1, len(cells)-len(first)-len(last))

	lines := []string{
		labelStyle.Render("Daily average"),
		sparkline(cells, passing),
		labelStyle.Render(first + strings.Repeat(" ", gap) + last),
		"",
		labelStyle.Render("Weekly"),
	}
	for _, wb := range weeklyBuckets(cells) {
		label := lipgloss.NewStyle().Foreground(colorMuted).Width(14).Render(rangeLabel(wb.Start, wb.End))
		if wb.Attempts == 0 {
			lines = append(lines, label+labelStyle.Render("no commits"))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s%s %s  %s", label, scoreBar(wb.Average, false),
			lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render(fmt.Sprintf("%.2f", wb.Average)),
			labelStyle.Render(fmt.Sprintf("%d commits", wb.Attempts))))
	}
	return lines
}
