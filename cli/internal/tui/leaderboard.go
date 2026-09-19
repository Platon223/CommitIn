package tui

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// LeaderboardRow is one ranked row.
type LeaderboardRow struct {
	Rank          int
	Username      string
	AverageScore  float64
	CommitCount   int
	IsCurrentUser bool
}

// LeaderboardInfo carries the ranking rules the backend applied, shown as
// context under the title.
type LeaderboardInfo struct {
	WindowDays int
	MinCommits int
}

const barWidth = 10

// scoreBar renders avg (0-10) as a fixed-width block bar. When plain is true
// no styling is applied, so it stays legible on the inverted current-user row.
func scoreBar(avg float64, plain bool) string {
	n := int(math.Round(avg))
	if n < 0 {
		n = 0
	}
	if n > barWidth {
		n = barWidth
	}
	filled, empty := strings.Repeat("█", n), strings.Repeat("░", barWidth-n)
	if plain {
		return filled + empty
	}
	return lipgloss.NewStyle().Foreground(colorAccent).Render(filled) +
		lipgloss.NewStyle().Foreground(colorMuted).Render(empty)
}

func renderLeaderboardTable(rows []LeaderboardRow) string {
	ink := lipgloss.Color("#000000")
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("RANK", "USER", "SCORE", "AVG", "COMMITS").
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)
			if col >= 3 {
				base = base.Align(lipgloss.Right)
			}
			switch {
			case row == table.HeaderRow:
				return base.Bold(true).Foreground(colorMuted)
			case row >= 0 && row < len(rows) && rows[row].IsCurrentUser:
				// Brand inversion: the logo's cream-on-black, flipped.
				return base.Bold(true).Foreground(ink).Background(colorAccent)
			case row >= 0 && row < 3 && col <= 1:
				return base.Bold(true).Foreground(colorAccent)
			default:
				return base.Foreground(colorMuted)
			}
		})
	for _, r := range rows {
		rank := strconv.Itoa(r.Rank)
		if r.IsCurrentUser {
			rank = "→ " + rank
		}
		t.Row(
			rank,
			r.Username,
			scoreBar(r.AverageScore, r.IsCurrentUser),
			strconv.FormatFloat(r.AverageScore, 'f', -1, 64),
			strconv.Itoa(r.CommitCount),
		)
	}
	return t.Render()
}

// renderLeaderboardBody is the title, rules line, and table. With width > 0
// the block is centered in that many columns; with width == 0 (non-TTY
// fallback) it's left-aligned.
func renderLeaderboardBody(rows []LeaderboardRow, info LeaderboardInfo, width int) string {
	tbl := renderLeaderboardTable(rows)
	title := titleStyle.Render("CommitIn Leaderboard")
	rules := labelStyle.Render(fmt.Sprintf("rolling %d-day average · minimum %d judged commits", info.WindowDays, info.MinCommits))

	// Center each element inside the widest one -- with short usernames the
	// subtitle is wider than the table, with long ones it's the reverse.
	blockWidth := max(lipgloss.Width(tbl), lipgloss.Width(title), lipgloss.Width(rules))
	center := func(s string) string { return lipgloss.PlaceHorizontal(blockWidth, lipgloss.Center, s) }

	body := center(title) + "\n" + center(rules) + "\n\n" + center(tbl)
	if width > 0 {
		body = lipgloss.PlaceHorizontal(width, lipgloss.Center, body)
	}
	return body
}

// yourStanding is the footer summary, or "" if the current user isn't on
// the board.
func yourStanding(rows []LeaderboardRow) string {
	for _, r := range rows {
		if r.IsCurrentUser {
			return fmt.Sprintf("You are #%d of %d", r.Rank, len(rows))
		}
	}
	return ""
}

type leaderboardModel struct {
	rows  []LeaderboardRow
	info  LeaderboardInfo
	vp    viewport.Model
	ready bool
}

func (m leaderboardModel) Init() tea.Cmd { return nil }

func (m leaderboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		const footerHeight = 2 // standing line + help line
		content := "\n" + renderLeaderboardBody(m.rows, m.info, msg.Width)
		if !m.ready {
			m.vp = viewport.New(msg.Width, msg.Height-footerHeight)
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = msg.Height - footerHeight
		}
		m.vp.SetContent(content)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m leaderboardModel) View() string {
	if !m.ready {
		return ""
	}
	standing := yourStanding(m.rows)
	if standing != "" {
		standing = lipgloss.PlaceHorizontal(m.vp.Width, lipgloss.Center, focusedLabelStyle.Render(standing))
	}
	help := lipgloss.PlaceHorizontal(m.vp.Width, lipgloss.Center, helpStyle.Render("↑/↓ scroll • q to quit"))
	return m.vp.View() + "\n" + standing + "\n" + help
}

// RunLeaderboard shows the ranked leaderboard as a full-screen, scrollable,
// centered view. If Bubble Tea can't start (no TTY -- piped/redirected
// output, e.g. `cmtin leaderboard | grep me`), it falls back to printing the
// body directly, left-aligned.
func RunLeaderboard(rows []LeaderboardRow, info LeaderboardInfo) {
	m := leaderboardModel{rows: rows, info: info}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(renderLeaderboardBody(rows, info, 0))
		if s := yourStanding(rows); s != "" {
			fmt.Println("\n" + s)
		}
	}
}

// PrintEmptyLeaderboard renders the "nobody's qualified yet" state as a
// small boxed message rather than a bare line.
func PrintEmptyLeaderboard(w io.Writer, info LeaderboardInfo) {
	body := titleStyle.Render("Nobody's on the board yet") + "\n" +
		labelStyle.Render(fmt.Sprintf("Get %d judged commits within %d days to be the first.", info.MinCommits, info.WindowDays))
	fmt.Fprintln(w, boxStyle(colorMuted).Render(body))
}
