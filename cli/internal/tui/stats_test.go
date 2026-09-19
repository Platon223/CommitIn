package tui

import (
	"bytes"
	"strings"
	"testing"
)

func TestRejectionRate(t *testing.T) {
	if got := (PeriodView{Attempts: 8, Rejected: 2}).rejectionRate(); got != 25 {
		t.Fatalf("got %v, want 25", got)
	}
	if got := (PeriodView{}).rejectionRate(); got != 0 {
		t.Fatalf("no attempts must not divide by zero, got %v", got)
	}
}

func TestFormatDelta(t *testing.T) {
	cases := []struct {
		name          string
		cur, prev     float64
		decimals      int
		unit          string
		kind          deltaKind
		wantSubstring string
	}{
		{"score up", 7, 5.33, 2, "", deltaHigherIsBetter, "▲ +1.67"},
		{"score down", 5, 6.5, 2, "", deltaHigherIsBetter, "▼ -1.50"},
		{"rejection down", 25, 66.7, 1, " pts", deltaLowerIsBetter, "▼ -41.7 pts"},
		{"volume up", 8, 3, 0, "", deltaNeutral, "▲ +5"},
		{"unchanged", 7, 7, 2, "", deltaHigherIsBetter, "no change"},
		{"rounds away to unchanged", 7.001, 7, 2, "", deltaHigherIsBetter, "no change"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatDelta(c.cur, c.prev, c.decimals, c.unit, c.kind); !strings.Contains(got, c.wantSubstring) {
				t.Fatalf("formatDelta = %q, want it to contain %q", got, c.wantSubstring)
			}
		})
	}
}

func TestRenderStatsWithComparison(t *testing.T) {
	out := renderStats(StatsView{
		WindowDays:   30,
		PassingScore: 6,
		Current:      PeriodView{Attempts: 8, Average: 7, Rejected: 2},
		Previous:     PeriodView{Attempts: 3, Average: 5.33, Rejected: 2},
	})
	for _, want := range []string{
		"Your CommitIn stats", "last 30 days, compared with the 30 before",
		"7.00 / 10", "▲ +1.67", // average improved
		"Judged commits", "▲ +5",
		"2 (25%)", "▼ -41.7 pts", // rejection rate fell 66.7% -> 25%
		"scored below 6",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stats card missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "no earlier data") {
		t.Fatalf("both windows have data, shouldn't say there's nothing to compare:\n%s", out)
	}
}

func TestRenderStatsWithoutPreviousWindow(t *testing.T) {
	out := renderStats(StatsView{
		WindowDays: 30, PassingScore: 6,
		Current: PeriodView{Attempts: 4, Average: 8, Rejected: 0},
	})
	if !strings.Contains(out, "8.00 / 10") || !strings.Contains(out, "0 (0%)") {
		t.Fatalf("current numbers missing:\n%s", out)
	}
	if got := strings.Count(out, "no earlier data to compare"); got != 3 {
		t.Fatalf("expected the no-comparison note on all 3 rows, got %d:\n%s", got, out)
	}
	if strings.Contains(out, "▲") || strings.Contains(out, "▼") {
		t.Fatalf("must not invent a trend against an empty window:\n%s", out)
	}
}

func TestPrintEmptyStats(t *testing.T) {
	var buf bytes.Buffer
	PrintEmptyStats(&buf, 30)
	for _, want := range []string{"No stats yet", "last 30 days", "cmtin init"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("empty state missing %q:\n%s", want, buf.String())
		}
	}
}
