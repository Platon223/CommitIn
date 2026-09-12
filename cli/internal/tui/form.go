package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ErrCancelled is returned by RunForm when the user quits (esc / ctrl+c).
var ErrCancelled = errors.New("cancelled")

// FormField describes one input in a RunForm form.
type FormField struct {
	Label       string
	Placeholder string
	Password    bool
}

type formModel struct {
	title  string
	fields []FormField
	inputs []textinput.Model
	focus  int
	err    error
}

// RunForm shows a multi-field form (tab / shift+tab to move between fields,
// enter on the last field submits, esc / ctrl+c cancels) and returns the
// entered values in the same order as fields.
func RunForm(title string, fields []FormField) ([]string, error) {
	m := newFormModel(title, fields)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		// The form was never rendered, so this is the only chance to tell
		// the user anything (e.g. no TTY available: signup/login need a
		// real terminal, same as most CLIs' interactive auth flows).
		fmt.Fprintf(os.Stderr, "could not start the interactive prompt: %v\n", err)
		return nil, err
	}
	fm := final.(formModel)
	if fm.err != nil {
		return nil, fm.err
	}
	values := make([]string, len(fm.inputs))
	for i, in := range fm.inputs {
		values[i] = strings.TrimSpace(in.Value())
	}
	return values, nil
}

func newFormModel(title string, fields []FormField) formModel {
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		ti.Prompt = ""
		ti.CharLimit = 256
		if f.Password {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}
	return formModel{title: title, fields: fields, inputs: inputs}
}

func (m formModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrCancelled
			return m, tea.Quit
		case tea.KeyEnter:
			if m.focus == len(m.inputs)-1 {
				return m, tea.Quit
			}
			m.nextInput()
		case tea.KeyTab, tea.KeyDown:
			m.nextInput()
		case tea.KeyShiftTab, tea.KeyUp:
			m.prevInput()
		}
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *formModel) nextInput() {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + 1) % len(m.inputs)
	m.inputs[m.focus].Focus()
}

func (m *formModel) prevInput() {
	m.inputs[m.focus].Blur()
	m.focus--
	if m.focus < 0 {
		m.focus = len(m.inputs) - 1
	}
	m.inputs[m.focus].Focus()
}

func (m formModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")
	for i, f := range m.fields {
		if i == m.focus {
			b.WriteString(focusedLabelStyle.Render("› " + f.Label))
		} else {
			b.WriteString("  " + labelStyle.Render(f.Label))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}
	b.WriteString(helpStyle.Render("tab/shift+tab move • enter next/submit • esc cancel"))
	return b.String()
}
