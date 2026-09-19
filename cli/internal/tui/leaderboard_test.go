package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func sampleRows() []LeaderboardRow {
	return []LeaderboardRow{
		{Rank: 1, Username: "alice", AverageScore: 9.1, CommitCount: 12},
		{Rank: 2, Username: "bob", AverageScore: 7.5, CommitCount: 20, IsCurrentUser: true},
		{Rank: 3, Username: "carol", AverageScore: 6, CommitCount: 30},
	}
}

func TestScoreBarIsAlwaysBarWidth(t *testing.T) {
	for _, avg := range []float64{-1, 0, 0.4, 5, 6.6, 9.1, 10, 99} {
		for _, plain := range []bool{true, false} {
			if got := lipgloss.Width(scoreBar(avg, plain)); got != barWidth {
				t.Fatalf("scoreBar(%v, plain=%v) is %d columns wide, want %d", avg, plain, got, barWidth)
			}
		}
	}
}

func TestScoreBarFillsProportionally(t *testing.T) {
	if got := strings.Count(scoreBar(7.4, true), "█"); got != 7 {
		t.Fatalf("7.4 should fill 7 blocks, got %d", got)
	}
}

func TestBodyContainsRowsRulesAndMarker(t *testing.T) {
	body := renderLeaderboardBody(sampleRows(), LeaderboardInfo{WindowDays: 30, MinCommits: 10}, 0)
	for _, want := range []string{
		"CommitIn Leaderboard",
		"rolling 30-day average",
		"minimum 10 judged commits",
		"alice", "bob", "carol",
		"→ 2", // the current user's row keeps a text marker even without color
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "→ 1") || strings.Contains(body, "→ 3") {
		t.Fatalf("marker leaked onto another row:\n%s", body)
	}
}

func TestBodyCentersOnlyWhenWidthGiven(t *testing.T) {
	info := LeaderboardInfo{WindowDays: 30, MinCommits: 10}

	// Width 0: the block itself is flush-left -- its widest line (the
	// subtitle here) starts in column 0, and nothing pokes out past it.
	left := renderLeaderboardBody(sampleRows(), info, 0)
	lines := strings.Split(left, "\n")
	blockWidth := 0
	for _, l := range lines {
		blockWidth = max(blockWidth, lipgloss.Width(l))
	}
	flush := false
	for _, l := range lines {
		if l != "" && !strings.HasPrefix(l, " ") {
			flush = true
		}
		if lipgloss.Width(l) > blockWidth {
			t.Fatalf("line wider than the block: %q", l)
		}
	}
	if !flush {
		t.Fatalf("width 0: no line starts in column 0, block isn't flush-left:\n%q", left)
	}

	centered := renderLeaderboardBody(sampleRows(), info, 140)
	for i, line := range strings.Split(centered, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			t.Fatalf("line %d not centered in 140 columns: %q", i, line)
		}
	}
}

func TestYourStanding(t *testing.T) {
	if got := yourStanding(sampleRows()); got != "You are #2 of 3" {
		t.Fatalf("got %q", got)
	}

	rows := sampleRows()
	rows[1].IsCurrentUser = false
	if got := yourStanding(rows); got != "" {
		t.Fatalf("expected no standing when the user isn't on the board, got %q", got)
	}
}

func TestPrintEmptyLeaderboardUsesBackendRules(t *testing.T) {
	var buf bytes.Buffer
	PrintEmptyLeaderboard(&buf, LeaderboardInfo{WindowDays: 14, MinCommits: 25})
	out := buf.String()
	if !strings.Contains(out, "25 judged commits within 14 days") {
		t.Fatalf("empty state should reflect the backend's rules, got:\n%s", out)
	}
}
