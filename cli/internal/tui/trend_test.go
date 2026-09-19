package tui

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func day(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestBuildDaySeriesIsContinuousAndOldestFirst(t *testing.T) {
	today := day("2026-09-19")
	cells := buildDaySeries([]DayView{
		{Date: "2026-09-19", Attempts: 2, Average: 6.5},
		{Date: "2026-08-20", Attempts: 1, Average: 10}, // 30 days back: the window's oldest, partial day
	}, today, 30)

	if len(cells) != 31 {
		t.Fatalf("the backend window touches 31 UTC dates, got %d cells", len(cells))
	}
	if got := cells[0].Date.Format(dateLayout); got != "2026-08-20" {
		t.Fatalf("first cell = %s", got)
	}
	if got := cells[30].Date.Format(dateLayout); got != "2026-09-19" {
		t.Fatalf("last cell = %s", got)
	}
	if !cells[0].hasData() || cells[0].Average != 10 || !cells[30].hasData() || cells[30].Attempts != 2 {
		t.Fatalf("data landed on the wrong cells: first=%+v last=%+v", cells[0], cells[30])
	}
	if cells[15].hasData() {
		t.Fatal("a day with no commits must stay empty, not zero")
	}
}

func TestBuildDaySeriesSurvivesClockSkew(t *testing.T) {
	// The server already rolled over to the 20th; the local clock hasn't.
	cells := buildDaySeries([]DayView{{Date: "2026-09-20", Attempts: 1, Average: 9}}, day("2026-09-19"), 30)
	last := cells[len(cells)-1]
	if last.Date.Format(dateLayout) != "2026-09-20" || !last.hasData() {
		t.Fatalf("a day the server reports must not be dropped, last cell = %+v", last)
	}
}

func TestSparklineGlyphsColorsAndGaps(t *testing.T) {
	cells := []dayCell{
		{Attempts: 1, Average: 10}, // tallest
		{Attempts: 1, Average: 0},  // lowest (still drawn: it's data)
		{},                         // no commits
		{Attempts: 1, Average: 5},  // below the pass mark
	}
	got := sparkline(cells, 6)
	if lipgloss.Width(got) != len(cells) {
		t.Fatalf("one column per day, got width %d for %d days", lipgloss.Width(got), len(cells))
	}
	// Colors are stripped in tests, so compare glyphs only. A 5 sits exactly
	// between glyphs 3 and 4 (3.5 on the 0-7 scale) and rounds half up to ▅.
	if want := "█▁·▅"; got != want {
		t.Fatalf("sparkline = %q, want %q", got, want)
	}
}

func TestWeeklyBucketsWeightByAttemptsAndPartialWeekIsOldest(t *testing.T) {
	cells := buildDaySeries(nil, day("2026-09-19"), 30) // 31 days
	// Newest full week (Sep 13-19): 1 attempt @10 and 3 @6 -> weighted 7, not the naive (10+6)/2 = 8.
	set := func(date string, attempts int, avg float64) {
		for i := range cells {
			if cells[i].Date.Format(dateLayout) == date {
				cells[i].Attempts, cells[i].Average = attempts, avg
			}
		}
	}
	set("2026-09-15", 1, 10)
	set("2026-09-17", 3, 6)

	wbs := weeklyBuckets(cells)
	if len(wbs) != 5 {
		t.Fatalf("31 days = 4 full weeks + a 3-day remainder, got %d buckets", len(wbs))
	}
	newest := wbs[len(wbs)-1]
	if newest.Start.Format(dateLayout) != "2026-09-13" || newest.End.Format(dateLayout) != "2026-09-19" {
		t.Fatalf("newest bucket must be a full week ending today, got %v-%v", newest.Start, newest.End)
	}
	if newest.Attempts != 4 || math.Abs(newest.Average-7) > 1e-9 {
		t.Fatalf("weighted average wrong: attempts=%d avg=%v, want 4 / 7", newest.Attempts, newest.Average)
	}
	oldest := wbs[0]
	if span := int(oldest.End.Sub(oldest.Start).Hours()/24) + 1; span != 3 {
		t.Fatalf("the partial bucket should be the oldest and hold 3 days, got %d", span)
	}
	if oldest.Attempts != 0 || oldest.Average != 0 {
		t.Fatalf("empty bucket should be zeroed, got %+v", oldest)
	}
}

func TestRangeLabel(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"2026-09-13", "2026-09-19", "Sep 13–19"},
		{"2026-08-30", "2026-09-05", "Aug 30–Sep 5"},
		{"2026-09-19", "2026-09-19", "Sep 19"},
	}
	for _, c := range cases {
		if got := rangeLabel(day(c.a), day(c.b)); got != c.want {
			t.Fatalf("rangeLabel(%s,%s) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

func TestStatsCardIncludesChartOnlyWithData(t *testing.T) {
	base := StatsView{
		WindowDays: 30, PassingScore: 6,
		Current:  PeriodView{Attempts: 3, Average: 7, Rejected: 1},
		Previous: PeriodView{Attempts: 2, Average: 6, Rejected: 1},
	}

	without := renderStats(base)
	if strings.Contains(without, "Daily average") || strings.Contains(without, "chart:") {
		t.Fatalf("no daily data, so no chart or legend:\n%s", without)
	}

	base.Today = day("2026-09-19")
	base.Daily = []DayView{{Date: "2026-09-19", Attempts: 3, Average: 7}}
	with := renderStats(base)
	for _, want := range []string{"Daily average", "Aug 20", "Sep 19", "Weekly", "Sep 13–19", "7.00", "3 commits", "no commits", "chart:"} {
		if !strings.Contains(with, want) {
			t.Fatalf("chart card missing %q:\n%s", want, with)
		}
	}
}
