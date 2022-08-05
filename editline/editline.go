package editline

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbles/textarea"
	"github.com/muesli/termenv"
)

// Model represents a widget that supports multi-line entry with
// auto-growing of the text height.
type Model struct {
	// CheckInputComplete is called when the Enter key is pressed.  It
	// determines whether a newline character should be added to the
	// input (callback returns false) or whether the input should
	// terminate altogether (callback returns true).  The callback is
	// provided the text of the input and the line number at which the
	// cursor is currently positioned as argument.
	CheckInputComplete func(string, int) bool

	p *tea.Program

	text      textarea.Model
	maxHeight int
	err       error
}

// New creates a new Model. The caller is responsible for calling
// SetProgram() after New.
func New() *Model {
	m := Model{
		text: textarea.New(),
		err:  nil,
	}
	m.CheckInputComplete = func(v string, row int) bool {
		vs := strings.Split(v, "\n")
		if row == len(vs)-1 && // Enter on last row.
			strings.HasSuffix(v, ";") { // Semicolon on last row.
			return true
		}
		return false
	}
	// Width will be set by Update below on init.
	m.text.Prompt = "> "
	m.text.Placeholder = ""
	m.text.ShowLineNumbers = false
	m.text.SetHeight(1)
	m.text.Focus()
	m.text.KeyMap.Paste.Unbind() // paste from clipboard not supported.
	return &m
}

// SetProgram must be called after New to attach the model to a
// Program. This is used so that the model can learn how to clear the
// screen.
func (m *Model) SetProgram(p *tea.Program) {
	m.p = p
}

func (m *Model) Err() error {
	return m.err
}

func (m *Model) Value() string {
	return m.text.Value()
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.text.SetWidth(msg.Width - 1)
		m.maxHeight = msg.Height
		h := clamp(m.text.Height(), 1, m.maxHeight-1)
		m.text.SetHeight(h)

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyCtrlBackslash:
			// FIXME: support sending SIGQUIT to process.

		case tea.KeyCtrlL:
			_ = m.p.ReleaseTerminal()
			termenv.ClearScreen()
			_ = m.p.RestoreTerminal()

		case tea.KeyCtrlD:
			if m.text.AtBeginningOfLine() {
				// FIXME: support returning io.EOF.
				return m, tea.Quit
			}

		case tea.KeyCtrlC:
			if m.text.EmptyValue() {
				// FIXME: support returning an error.
				return m, tea.Quit
			}
			m.text.SetValue("")
			return m, nil

		case tea.KeyEnter:
			if m.CheckInputComplete != nil {
				if m.CheckInputComplete(m.text.Value(), m.text.Line()) {
					return m, tea.Quit
				}
			}
		}
	}

	m.text, cmd = m.text.Update(msg)

	m.text.SetHeight(clamp(m.computeLogicalHeight()+1, 1, m.maxHeight-1))

	return m, cmd
}

func (m Model) computeLogicalHeight() int {
	logicalHeight := 0
	nl := m.text.NumLinesInValue()
	for row := 0; row < nl; row++ {
		li := m.text.LineInfoAt(row, 0)
		logicalHeight += li.Height
	}
	return logicalHeight
}

func (m Model) View() string {
	return m.text.View() // + "\n"
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
