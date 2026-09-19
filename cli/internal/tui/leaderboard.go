package tui

import (
	"fmt"
	"strconv"

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

func renderLeaderboardTable(rows []LeaderboardRow) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("RANK", "USERNAME", "AVG SCORE", "COMMITS").
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)
			switch {
			case row == table.HeaderRow:
				return base.Bold(true).Foreground(colorAccent)
			case row >= 0 && row < len(rows) && rows[row].IsCurrentUser:
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
		t.Row(rank, r.Username, strconv.FormatFloat(r.AverageScore, 'f', -1, 64), strconv.Itoa(r.CommitCount))
	}
	return t.Render()
}

type leaderboardModel struct {
	content string
	vp      viewport.Model
	ready   bool
}

func (m leaderboardModel) Init() tea.Cmd { return nil }

func (m leaderboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		const helpHeight = 1
		if !m.ready {
			m.vp = viewport.New(msg.Width, msg.Height-helpHeight)
			m.vp.SetContent(m.content)
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = msg.Height - helpHeight
		}
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
	return m.vp.View() + "\n" + helpStyle.Render("↑/↓ scroll • q to quit")
}

// RunLeaderboard shows the ranked leaderboard as a full-screen, scrollable
// view. If Bubble Tea can't start (no TTY -- piped/redirected output, e.g.
// `cmtin leaderboard | grep me`), it falls back to printing the table
// directly.
func RunLeaderboard(rows []LeaderboardRow) {
	content := renderLeaderboardTable(rows)
	m := leaderboardModel{content: content}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(content)
	}
}
