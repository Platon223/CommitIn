package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerDoneMsg struct{}

func waitForDone(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-done
		return spinnerDoneMsg{}
	}
}

type spinnerModel struct {
	label   string
	spinner spinner.Model
	done    <-chan struct{}
	stopped bool
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForDone(m.done))
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(spinnerDoneMsg); ok {
		m.stopped = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.stopped {
		// Empty final frame: Bubble Tea's renderer erases the previous
		// (shorter) frame, so the spinner line disappears rather than
		// freezing mid-spin -- the caller prints its own result right after.
		return ""
	}
	return m.spinner.View() + " " + m.label
}

// RunSpinner shows a spinner next to label while work runs in the
// background, returning once work completes. Unlike RunTask, it renders no
// success/failure box -- callers print their own result afterward with
// PrintSuccess/PrintError. work always runs to completion exactly once, even
// if Bubble Tea itself can't start (e.g. no TTY): in that case RunSpinner
// just waits for it silently instead of showing a spinner.
func RunSpinner(label string, work func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		work()
	}()

	m := spinnerModel{label: label, spinner: newSpinner(), done: done}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		<-done
	}
}
