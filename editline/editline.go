package editline

import (
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/editline/internal/textarea"
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

	// MaxHistorySize is the maximum number of entries in the history.
	// Set to zero for no limit.
	MaxHistorySize int

	// DedupHistory if true avoids adding a history entry
	// if it is equal to the last one added.
	DedupHistory bool

	// Prompt is the prompt displayed before entry lines.
	// Only takes effect at Reset().
	Prompt string

	// TODO: separate 1st line prompt vs all-but-first-line prompt.

	// SearchPrompt is the prompt displayed before the history search pattern.
	SearchPrompt string
	// SearchPromptNotFound is the prompt displayed before the history search pattern,
	// when no match is found.
	SearchPromptNotFound string

	// ShowLineNumbers if true shows line numbers at the beginning
	// of each input line.
	// Only takes effect at Reset().
	ShowLineNumbers bool

	history []string
	hctrl   struct {
		// searching is true when we're currently searching.
		searching   bool
		pattern     textinput.Model
		prevPattern string
		cursor      int
		// value prior to the search starting.
		valueSaved bool
		prevValue  string
		prevCursor int
	}

	p *tea.Program

	text      textarea.Model
	maxHeight int
}

// New creates a new Model. The caller is responsible for calling
// SetProgram() after New.
func New() *Model {
	m := Model{
		text:                 textarea.New(),
		MaxHistorySize:       0, // no limit
		DedupHistory:         true,
		Prompt:               "> ",
		SearchPrompt:         "bck:",
		SearchPromptNotFound: "bck?",
		ShowLineNumbers:      false,
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
	m.text.KeyMap.Paste.Unbind() // paste from clipboard not supported.
	m.hctrl.pattern = textinput.New()
	m.text.Placeholder = ""
	m.Reset()
	return &m
}

// SetHistory sets the history navigation list all at once.
func (m *Model) SetHistory(h []string) {
	if m.MaxHistorySize != 0 && len(h) > m.MaxHistorySize {
		h = h[:m.MaxHistorySize]
	}
	m.history = make([]string, 0, len(h))
	for _, e := range h {
		m.history = append(m.history, e)
	}
	m.resetNavCursor()
}

// GetHistory retrieves all the entries in the history navigation list.
func (m *Model) GetHistory() []string {
	return m.history
}

// AddHistoryEntry adds an entry to the history navigation list.
func (m *Model) AddHistoryEntry(s string) {
	// Only add a new entry if it doesn't duplicate the last one.
	if len(m.history) == 0 || !(m.DedupHistory && s == m.history[len(m.history)-1]) {
		m.history = append(m.history, s)
	}
	// Truncate if needed.
	if m.MaxHistorySize != 0 && len(m.history) > m.MaxHistorySize {
		copy(m.history, m.history[1:])
		m.history = m.history[:m.MaxHistorySize]
	}
	m.resetNavCursor()
}

// SetProgram must be called after New to attach the model to a
// Program. This is used so that the model can learn how to clear the
// screen.
func (m *Model) SetProgram(p *tea.Program) {
	m.p = p
}

// Value retrieves the value of the text input.
func (m *Model) Value() string {
	return m.text.Value()
}

// Init is part of the tea.Model interface.
func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) historyStartSearch() {
	m.hctrl.searching = true
	m.hctrl.pattern.Prompt = m.SearchPrompt
	m.hctrl.pattern.Reset()
	m.hctrl.pattern.Focus()
	m.hctrl.prevPattern = ""
	m.saveValue()
	m.resetNavCursor()
}

func (m *Model) resetNavCursor() {
	m.hctrl.cursor = len(m.history)
}

func (m *Model) cancelHistorySearch() {
	m.hctrl.searching = false
	m.hctrl.pattern.Blur()
	m.restoreValue()
	m.text.Focus()
}

func (m *Model) restoreValue() {
	m.updateValue(m.hctrl.prevValue, m.hctrl.prevCursor)
	m.hctrl.valueSaved = false
	m.hctrl.prevValue = ""
	m.hctrl.prevCursor = 0
	m.resetNavCursor()
}

func (m *Model) acceptSearch() {
	m.hctrl.searching = false
	m.hctrl.pattern.Blur()
	m.hctrl.valueSaved = false
	m.hctrl.prevValue = ""
	m.hctrl.prevCursor = 0
	// entry := m.history[m.hctrl.cursor]
	// m.updateValue(entry, len(entry))
	m.text.Focus()
}

func (m *Model) incrementalSearch(nextMatch bool) {
	pat := m.hctrl.pattern.Value()
	if pat == m.hctrl.prevPattern {
		if !nextMatch {
			// Nothing changed, and no request for incremental search: do nothing.
			return
		}
		// Just nextMatch: continue incremental search below.
	} else {
		// Pattern changed, start again.
		m.resetNavCursor()
		m.hctrl.prevPattern = pat
	}

	i := m.hctrl.cursor - 1
	for ; i >= 0; i-- {
		entry := m.history[i]
		for j := len(entry) - len(pat); j >= 0; j-- {
			// TODO: use glob search instead of simple strings.
			if strings.HasPrefix(entry[j:], pat) {
				// It's a match!
				m.hctrl.pattern.Prompt = m.SearchPrompt
				m.hctrl.cursor = i
				m.updateValue(entry, j)
				return
			}
		}
	}
	if i < 0 {
		// No match found.
		m.hctrl.pattern.Prompt = m.SearchPromptNotFound
	}
}

func (m *Model) updateValue(value string, cursor int) {
	m.text.SetValue(value)
	m.text.SetCursor(cursor)
	m.updateTextSz()
	m.text.ResetViewCursorDown()
}

func (m *Model) updateTextSz() {
	m.text.SetHeight(clamp(m.text.LogicalHeight(), 1, m.maxHeight-1))
}

func (m *Model) saveValue() {
	m.hctrl.valueSaved = true
	m.hctrl.prevValue = m.text.Value()
	m.hctrl.prevCursor = m.text.CursorPos()
}

func (m *Model) historyUp() {
	if !m.hctrl.valueSaved {
		m.saveValue()
	}
	if m.hctrl.cursor == 0 {
		return
	}
	m.hctrl.cursor--
	entry := m.history[m.hctrl.cursor]
	m.updateValue(entry, len(entry))
}

func (m *Model) historyDown() {
	if !m.hctrl.valueSaved {
		m.saveValue()
	}
	m.hctrl.cursor++
	if m.hctrl.cursor >= len(m.history) {
		m.restoreValue()
		return
	}
	entry := m.history[m.hctrl.cursor]
	m.updateValue(entry, len(entry))
}

// Update is the Bubble Tea event handler.
// This is part of the tea.Model interface.
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
			pr, err := os.FindProcess(os.Getpid())
			if err != nil {
				// No-op.
				return m, nil
			}
			_ = m.p.ReleaseTerminal()
			pr.Signal(syscall.SIGQUIT)
			_ = m.p.RestoreTerminal()
			return m, nil

		case tea.KeyCtrlZ:
			pr, err := os.FindProcess(os.Getpid())
			if err != nil {
				// No-op.
				return m, nil
			}
			_ = m.p.ReleaseTerminal()
			pr.Signal(syscall.SIGTSTP)
			_ = m.p.RestoreTerminal()
			return m, nil

		case tea.KeyCtrlL:
			_ = m.p.ReleaseTerminal()
			termenv.ClearScreen()
			_ = m.p.RestoreTerminal()
			return m, nil

		case tea.KeyCtrlD:
			if m.text.AtBeginningOfLine() {
				if m.hctrl.searching {
					m.cancelHistorySearch()
				}
				// FIXME: support returning io.EOF.
				m.resetNavCursor()
				m.text.Blur()
				return m, tea.Quit
			}

		case tea.KeyCtrlG:
			if m.hctrl.searching {
				m.cancelHistorySearch()
				return m, nil
			}

		case tea.KeyCtrlR:
			if m.hctrl.searching {
				m.incrementalSearch(true /* nextMatch */)
			} else {
				// Start completion.
				m.historyStartSearch()
			}
			return m, nil

		case tea.KeyLeft, tea.KeyRight:
			if m.hctrl.searching {
				m.acceptSearch()
			}

		case tea.KeyUp:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			if m.text.AtFirstLineOfInputAndView() {
				m.historyUp()
				return m, nil
			}
			// Otherwise, fall through.

		case tea.KeyDown:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			if m.text.AtLastLineOfInputAndView() {
				m.historyDown()
				return m, nil
			}
			// Otherwise, fall through.

		case tea.KeyCtrlC:
			if m.text.EmptyValue() {
				// FIXME: support returning an error.
				m.resetNavCursor()
				m.text.Blur()
				return m, tea.Quit
			}
			if m.hctrl.searching {
				// Stop the completion. Do nothing further.
				m.cancelHistorySearch()
			} else {
				m.text.SetValue("")
			}
			return m, nil

		case tea.KeyEnter:
			if m.hctrl.searching {
				// Stop the completion first.
				m.acceptSearch()
			}
			if m.CheckInputComplete != nil {
				if m.CheckInputComplete(m.text.Value(), m.text.Line()) {
					// Reset the search/history navigation cursor to the end.
					m.resetNavCursor()
					m.text.Blur()
					return m, tea.Quit
				}
			}
		}
	}

	if m.hctrl.searching {
		// Add text to the pattern. Also incremental search.
		m.hctrl.pattern, cmd = m.hctrl.pattern.Update(msg)
		m.incrementalSearch(false /* nextMatch */)
	} else {
		m.text, cmd = m.text.Update(msg)
		m.updateTextSz()
	}

	return m, cmd
}

// Reset sets the input to its default state with no input.
// The history is preserved.
func (m *Model) Reset() {
	m.hctrl.valueSaved = false
	m.hctrl.prevValue = ""
	m.hctrl.prevCursor = 0
	m.text.ShowLineNumbers = m.ShowLineNumbers
	m.text.Prompt = m.Prompt
	m.text.SetHeight(1)
	m.text.Reset()
	m.text.Focus()
}

// View renders the text area in its current state.
// This is part of the tea.Model interface.
func (m Model) View() string {
	view := m.text.View()
	if m.hctrl.searching {
		view += "\n" + m.hctrl.pattern.View()
	}
	return view
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
