package editline

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/editline/internal/textarea"
	"github.com/muesli/termenv"
)

// ErrInterrupted is returned when the input is terminated
// with Ctrl+C.
var ErrInterrupted = errors.New("interrupted")

// Model represents a widget that supports multi-line entry with
// auto-growing of the text height.
type Model struct {
	// Err is the final state at the end of input.
	// Likely io.EOF or ErrInterrupted.
	Err error

	// CheckInputComplete is called when the Enter key is pressed.  It
	// determines whether a newline character should be added to the
	// input (callback returns false) or whether the input should
	// terminate altogether (callback returns true). The callback is
	// provided the text of the input and the line number at which the
	// cursor is currently positioned as argument.
	//
	// The default behavior if CheckInputComplete is nil is to terminate
	// the input when enter is pressed.
	CheckInputComplete func(entireInput string, cursorRow int) bool

	// AutoComplete if set is called upon the user pressing the
	// autocomplete key. The callback is provided the text of the input
	// and the position of the cursor in the input. The returned
	// extraInput value is added at the cursor position. The returned
	// msg is printed above the input box.
	AutoComplete func(entireInput string, cursor int) (msg, extraInput string)

	// MaxHistorySize is the maximum number of entries in the history.
	// Set to zero for no limit.
	MaxHistorySize int

	// DedupHistory if true avoids adding a history entry
	// if it is equal to the last one added.
	DedupHistory bool

	// Prompt is the prompt displayed before entry lines.
	// Only takes effect at Reset().
	Prompt string

	// NextPrompt, if defined is the prompt displayed before entry lines
	// after the first one.
	// Only takes effect at Reset().
	NextPrompt string

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

	text      textarea.Model
	maxHeight int
}

// New creates a new Model.
func New() *Model {
	m := Model{
		text:                 textarea.New(),
		Err:                  nil,
		MaxHistorySize:       0, // no limit
		DedupHistory:         true,
		Prompt:               "> ",
		NextPrompt:           "",
		SearchPrompt:         "bck:",
		SearchPromptNotFound: "bck?",
		ShowLineNumbers:      false,
	}
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

// Value retrieves the value of the text input.
func (m *Model) Value() string {
	val := m.text.Value()
	if len(val) > 0 && val[len(val)-1] == '\n' {
		// Trim final newline.
		val = val[:len(val)-1]
	}
	return val
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

func (m *Model) cancelHistorySearch() (cmd tea.Cmd) {
	m.hctrl.searching = false
	m.hctrl.pattern.Blur()
	cmd = m.restoreValue()
	m.text.Focus()
	return cmd
}

func (m *Model) restoreValue() (cmd tea.Cmd) {
	cmd = m.updateValue(m.hctrl.prevValue, m.hctrl.prevCursor)
	m.hctrl.valueSaved = false
	m.hctrl.prevValue = ""
	m.hctrl.prevCursor = 0
	m.resetNavCursor()
	return cmd
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

func (m *Model) incrementalSearch(nextMatch bool) (cmd tea.Cmd) {
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
				return m.updateValue(entry, j)
			}
		}
	}
	if i < 0 {
		// No match found.
		m.hctrl.pattern.Prompt = m.SearchPromptNotFound
	}
	return cmd
}

func (m *Model) updateValue(value string, cursor int) (cmd tea.Cmd) {
	m.text.SetValue(value)
	m.text.SetCursor(cursor)
	cmd = m.updateTextSz()
	return cmd
}

func (m *Model) updateTextSz() (cmd tea.Cmd) {
	newHeight := clamp(m.text.LogicalHeight(), 1, m.maxHeight-1)
	if m.text.Height() != newHeight {
		m.text.SetHeight(newHeight)
		if m.text.Line() == m.text.LineCount()-1 {
			m.text.ResetViewCursorDown()
		}
		// Process an empty event to reposition the cursor to a good place.
		m.text, cmd = m.text.Update(nil)
	}
	return cmd
}

func (m *Model) saveValue() {
	m.hctrl.valueSaved = true
	m.hctrl.prevValue = m.text.Value()
	m.hctrl.prevCursor = m.text.CursorPos()
}

func (m *Model) historyUp() (cmd tea.Cmd) {
	if !m.hctrl.valueSaved {
		m.saveValue()
	}
	if m.hctrl.cursor == 0 {
		return cmd
	}
	m.hctrl.cursor--
	entry := m.history[m.hctrl.cursor]
	return tea.Batch(cmd, m.updateValue(entry, len(entry)))
}

func (m *Model) historyDown() (cmd tea.Cmd) {
	if !m.hctrl.valueSaved {
		m.saveValue()
	}
	m.hctrl.cursor++
	if m.hctrl.cursor >= len(m.history) {
		return m.restoreValue()
	}
	entry := m.history[m.hctrl.cursor]
	return tea.Batch(cmd, m.updateValue(entry, len(entry)))
}

type doProgram func()

// Run is part of the tea.ExecCommand interface.
func (d doProgram) Run() error {
	d()
	return nil
}

// SetStdin is part of the tea.ExecCommand interface.
func (d doProgram) SetStdin(io.Reader) {}

// SetStdout is part of the tea.ExecCommand interface.
func (d doProgram) SetStdout(io.Writer) {}

// SetStderr is part of the tea.ExecCommand interface.
func (d doProgram) SetStderr(io.Writer) {}

// Update is the Bubble Tea event handler.
// This is part of the tea.Model interface.
func (m *Model) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	stop := false

	switch msg := imsg.(type) {
	case tea.WindowSizeMsg:
		m.text.SetWidth(msg.Width - 1)
		m.maxHeight = msg.Height
		cmd = m.updateTextSz()

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyTab:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			if m.AutoComplete != nil {
				msgs, extra := m.AutoComplete(m.text.Value(), m.text.CursorPos())
				m.text.InsertString(extra)
				cmd = m.updateTextSz()
				if msgs != "" {
					cmd = tea.Batch(cmd, tea.Println(msgs))
				}
				imsg = nil // consume message.
			}

		case tea.KeyCtrlBackslash:
			return m, tea.Exec(doProgram(func() {
				pr, err := os.FindProcess(os.Getpid())
				if err != nil {
					// No-op.
					return
				}
				pr.Signal(syscall.SIGQUIT)
			}), nil)

		case tea.KeyCtrlZ:
			return m, tea.Exec(doProgram(func() {
				pr, err := os.FindProcess(os.Getpid())
				if err != nil {
					// No-op.
					return
				}
				pr.Signal(syscall.SIGTSTP)
			}), nil)

		case tea.KeyCtrlL:
			return m, tea.Exec(doProgram(termenv.ClearScreen), nil)

		case tea.KeyCtrlD:
			if m.text.AtBeginningOfLine() {
				if m.hctrl.searching {
					cmd = m.cancelHistorySearch()
				}
				m.Err = io.EOF
				stop = true
				imsg = nil // consume message
			}

		case tea.KeyCtrlG:
			if m.hctrl.searching {
				cmd = m.cancelHistorySearch()
				imsg = nil // consume message
			}

		case tea.KeyCtrlR:
			if m.hctrl.searching {
				m.incrementalSearch(true /* nextMatch */)
			} else {
				// Start completion.
				m.historyStartSearch()
			}
			imsg = nil // consume message

		case tea.KeyLeft, tea.KeyRight:
			if m.hctrl.searching {
				m.acceptSearch()
			}

		case tea.KeyCtrlP:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			m.historyUp()
			imsg = nil // consume message

		case tea.KeyUp:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			if m.text.AtFirstLineOfInputAndView() {
				m.historyUp()
				imsg = nil // consume message
			}

		case tea.KeyCtrlN:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			m.historyDown()
			imsg = nil // consume message

		case tea.KeyDown:
			if m.hctrl.searching {
				m.acceptSearch()
			}
			if m.text.AtLastLineOfInputAndView() {
				m.historyDown()
				imsg = nil // consume message
			}

		case tea.KeyCtrlC:
			imsg = nil // consume message
			if m.text.EmptyValue() {
				m.Err = ErrInterrupted
				stop = true
				break
			}
			if m.hctrl.searching {
				// Stop the completion. Do nothing further.
				cmd = m.cancelHistorySearch()
			} else {
				m.text.SetValue("")
			}

		case tea.KeyEnter:
			if m.hctrl.searching {
				// Stop the completion first.
				m.acceptSearch()
			}
			if m.CheckInputComplete == nil ||
				m.CheckInputComplete(m.text.Value(), m.text.Line()) {
				stop = true
				// Fallthrough: we want the enter key to be processed
				// so that there's a final empty line in the display.
			}
		}
	}

	if m.hctrl.searching {
		// Add text to the pattern. Also incremental search.
		var newCmd tea.Cmd
		m.hctrl.pattern, newCmd = m.hctrl.pattern.Update(imsg)
		cmd = tea.Batch(cmd, newCmd,
			m.incrementalSearch(false /* nextMatch */))
	} else {
		var newCmd tea.Cmd
		m.text, newCmd = m.text.Update(imsg)
		cmd = tea.Batch(cmd, newCmd, m.updateTextSz())
	}

	if stop {
		// Reset the search/history navigation cursor to the end.
		m.resetNavCursor()
		m.text.Blur()
		cmd = tea.Batch(cmd, tea.Quit)
	}

	return m, cmd
}

// Reset sets the input to its default state with no input.
// The history is preserved.
func (m *Model) Reset() {
	m.Err = nil
	m.hctrl.valueSaved = false
	m.hctrl.prevValue = ""
	m.hctrl.prevCursor = 0
	m.text.ShowLineNumbers = m.ShowLineNumbers
	m.text.Prompt = m.Prompt
	m.text.NextPrompt = m.NextPrompt
	// Width will be set by Update below on init.
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
