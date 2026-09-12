package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type taskDoneMsg struct {
	lines []string
	err   error
}

type taskModel struct {
	label   string
	work    func() ([]string, error)
	spinner spinner.Model
	done    bool
	lines   []string
	err     error
}

// RunTask shows a spinner next to label while work runs in the background,
// then replaces it with a styled box: the lines work returns on success, or
// its error on failure. The final box stays on screen after the program
// exits. RunTask returns work's error, if any, so the caller can propagate
// it as the command's exit status.
func RunTask(label string, work func() ([]string, error)) error {
	m := taskModel{label: label, work: work, spinner: newSpinner()}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		// Nothing was rendered in this case (e.g. no TTY available), unlike
		// a work failure, which the box above already displayed.
		fmt.Fprintf(os.Stderr, "could not start the interactive UI: %v\n", err)
		return err
	}
	return final.(taskModel).err
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return s
}

func (m taskModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, runWork(m.work))
}

func runWork(work func() ([]string, error)) tea.Cmd {
	return func() tea.Msg {
		lines, err := work()
		return taskDoneMsg{lines: lines, err: err}
	}
}

func (m taskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case taskDoneMsg:
		m.done = true
		m.lines = msg.lines
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.err = ErrCancelled
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m taskModel) View() string {
	if !m.done {
		return m.spinner.View() + " " + m.label
	}
	if m.err != nil {
		return boxStyle(colorBad).Render(errStyle.Render("✗ " + m.err.Error()))
	}
	var b strings.Builder
	for i, l := range m.lines {
		if i == 0 {
			b.WriteString(successStyle.Render("✓ " + l))
			continue
		}
		b.WriteString("\n")
		b.WriteString(labelStyle.Render(l))
	}
	return boxStyle(colorGood).Render(b.String())
}
