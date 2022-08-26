package editline

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knz/bubbline/editline/internal/textarea"
	rw "github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

// ErrInterrupted is returned when the input is terminated
// with Ctrl+C.
var ErrInterrupted = errors.New("interrupted")

// Style that will be applied to the editor.
type Style struct {
	Editor textarea.Style

	SearchInput struct {
		PromptStyle      lipgloss.Style
		TextStyle        lipgloss.Style
		BackgroundStyle  lipgloss.Style
		PlaceholderStyle lipgloss.Style
		CursorStyle      lipgloss.Style
	}
}

// DefaultStyles returns the default styles for focused and blurred states for
// the textarea.
func DefaultStyles() (Style, Style) {
	ts1, ts2 := textarea.DefaultStyles()
	fs := Style{Editor: ts1}
	bs := Style{Editor: ts2}
	fs.SearchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return fs, bs
}

// KeyMap is the key bindings for actions within the editor.
type KeyMap struct {
	textarea.KeyMap

	EndOfInput      key.Binding
	Interrupt       key.Binding
	AutoComplete    key.Binding
	SignalQuit      key.Binding
	SignalTTYStop   key.Binding
	Refresh         key.Binding
	AbortSearch     key.Binding
	SearchBackward  key.Binding
	HistoryPrevious key.Binding
	HistoryNext     key.Binding
	Debug           key.Binding
	HideShowPrompt  key.Binding
	AlwaysNewline   key.Binding
}

// DefaultKeyMap is the default set of key bindings.
var DefaultKeyMap = KeyMap{
	KeyMap:          textarea.DefaultKeyMap,
	AlwaysNewline:   key.NewBinding(key.WithKeys("alt+enter", "alt+\r")),
	AutoComplete:    key.NewBinding(key.WithKeys("tab")),
	Interrupt:       key.NewBinding(key.WithKeys("ctrl+c")),
	SignalQuit:      key.NewBinding(key.WithKeys(`ctrl+\`)),
	SignalTTYStop:   key.NewBinding(key.WithKeys("ctrl+z")),
	Refresh:         key.NewBinding(key.WithKeys("ctrl+l")),
	EndOfInput:      key.NewBinding(key.WithKeys("ctrl+d")),
	AbortSearch:     key.NewBinding(key.WithKeys("ctrl+g")),
	SearchBackward:  key.NewBinding(key.WithKeys("ctrl+r")),
	HistoryPrevious: key.NewBinding(key.WithKeys("alt+p")),
	HistoryNext:     key.NewBinding(key.WithKeys("alt+n")),
	HideShowPrompt:  key.NewBinding(key.WithKeys("alt+.")),
}

// FindWordStart is meant for use as a helper when implementing
// AutoComplete callbacks for the Model.AutoComplete field.
// Given AutoComplete's callback arguments, it searches
// and returns the start of the word that the cursor is currently
// on (as defined by the earliest character from the cursor
// that's not a whitespace) on the same line.
//
// NB: it does not cross line boundaries. The length in runes
// of the prefix from the cursor to the beginning of the word
// is guaranteed to be col-wordStart.
func FindWordStart(v [][]rune, line, col int) (word string, wordStart int) {
	wordStart = col
	if wordStart > 0 && wordStart >= len(v[line]) {
		wordStart = len(v[line]) - 1
	}
	if wordStart > 0 && !unicode.IsSpace(v[line][wordStart]) {
		// Find beginning of word.
		for wordStart > 0 && !unicode.IsSpace(v[line][wordStart-1]) {
			wordStart--
		}
	}
	word = string(v[line][wordStart:col])
	return word, wordStart
}

// FindLongestCommonPrefix returns the longest common
// prefix between the two arguments.
func FindLongestCommonPrefix(first, last string) string {
	en := len(first)
	if len(last) < en {
		en = len(last)
	}
	i := 0
	for {
		r, w := utf8.DecodeRuneInString(first[i:])
		l, _ := utf8.DecodeRuneInString(last[i:])
		if i >= en || r != l {
			break
		}
		i += w
	}
	return first[:i]
}

// Model represents a widget that supports multi-line entry with
// auto-growing of the text height.
type Model struct {
	// Err is the final state at the end of input.
	// Likely io.EOF or ErrInterrupted.
	Err error

	// KeyMap is the key bindings to use.
	KeyMap KeyMap

	// Styling. FocusedStyle and BlurredStyle are used to style the textarea in
	// focused and blurred states.
	// Only takes effect at Reset() or Focus().
	FocusedStyle Style
	BlurredStyle Style

	// Placeholder is displayed when the editor is still empty.
	// Only takes effect at Reset() or Focus().
	Placeholder string

	// CheckInputComplete is called when the Enter key is pressed.  It
	// determines whether a newline character should be added to the
	// input (callback returns false) or whether the input should
	// terminate altogether (callback returns true). The callback is
	// provided the text of the input and the line number at which the
	// cursor is currently positioned as argument.
	//
	// The default behavior if CheckInputComplete is nil is to terminate
	// the input when enter is pressed.
	CheckInputComplete func(entireInput [][]rune, line, col int) bool

	// AutoComplete if set is called upon the user pressing the
	// autocomplete key. The callback is provided the text of the input
	// and the position of the cursor in the input.
	// The returned msg is printed above the input box.
	//
	// If the consumedChars is non-zero, that number of characters
	// is erased before the cursor.
	// Then the first string in the returned extraInput value is added at the cursor position.
	// If there is more than 1 entry in the returned extraInput, they are
	// displayed above the input box too.
	AutoComplete func(entireInput [][]rune, line, col int) (msg string, consumedChars int, extraInput []string)

	// CharLimit is the maximum size of the input in characters.
	// Set to zero or less for no limit.
	CharLimit int

	// MaxHistorySize is the maximum number of entries in the history.
	// Set to zero for no limit.
	MaxHistorySize int

	// DedupHistory if true avoids adding a history entry
	// if it is equal to the last one added.
	DedupHistory bool

	// DeleteCharIfNotEOF, if true, causes the EndOfInput key binding
	// to be translated to delete-character-forward when it is not
	// entered at the beginning of a line.
	// Meant for use when the EndOfInput key binding is Ctrl+D, which
	// is the standard character deletion in textarea/libedit.
	// This can be set to false if the EndOfInput binding is fully
	// separate from DeleteCharacterForward.
	DeleteCharIfNotEOF bool

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
	// Only takes effect at Reset() or Focus().
	ShowLineNumbers bool

	history []string
	hctrl   struct {
		pattern textinput.Model
		c       struct {
			// searching is true when we're currently searching.
			searching   bool
			prevPattern string
			cursor      int
			// value prior to the search starting.
			valueSaved bool
			prevValue  string
			prevCursor int
		}
	}
	hidePrompt bool

	text      textarea.Model
	maxWidth  int
	maxHeight int

	// Debugging data.
	debugMode bool
	lastEvent tea.Msg
}

// New creates a new Model.
func New() *Model {
	focusedStyle, blurredStyle := DefaultStyles()
	m := Model{
		text:                 textarea.New(),
		Err:                  nil,
		KeyMap:               DefaultKeyMap,
		MaxHistorySize:       0, // no limit
		DedupHistory:         true,
		DeleteCharIfNotEOF:   true,
		FocusedStyle:         focusedStyle,
		BlurredStyle:         blurredStyle,
		Placeholder:          "",
		Prompt:               "> ",
		NextPrompt:           "",
		SearchPrompt:         "bck:",
		SearchPromptNotFound: "bck?",
		ShowLineNumbers:      false,
	}
	m.text.KeyMap.Paste.Unbind() // paste from clipboard not supported.
	m.hctrl.pattern = textinput.New()
	m.hctrl.pattern.Placeholder = "enter search term, or ^G to cancel search"
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

// Focus sets the focus state on the model. When the model is in focus
// it can receive keyboard input and the cursor is displayed.
func (m *Model) Focus() tea.Cmd {
	m.text.KeyMap = m.KeyMap.KeyMap
	m.text.Placeholder = m.Placeholder
	m.text.ShowLineNumbers = m.ShowLineNumbers
	m.text.FocusedStyle = m.FocusedStyle.Editor
	m.text.BlurredStyle = m.BlurredStyle.Editor
	m.updatePrompt()
	m.hctrl.pattern.PromptStyle = m.FocusedStyle.SearchInput.PromptStyle
	m.hctrl.pattern.TextStyle = m.FocusedStyle.SearchInput.TextStyle
	m.hctrl.pattern.BackgroundStyle = m.FocusedStyle.SearchInput.BackgroundStyle
	m.hctrl.pattern.PlaceholderStyle = m.FocusedStyle.SearchInput.PlaceholderStyle
	m.hctrl.pattern.CursorStyle = m.FocusedStyle.SearchInput.CursorStyle

	var cmd tea.Cmd
	if m.hctrl.c.searching {
		cmd = m.hctrl.pattern.Focus()
	}
	return tea.Batch(cmd, m.text.Focus())
}

// Blur removes the focus state on the model. When the model is
// blurred it can not receive keyboard input and the cursor will be
// hidden.
func (m *Model) Blur() {
	m.hctrl.pattern.Blur()
	m.text.Blur()
	m.hctrl.pattern.PromptStyle = m.BlurredStyle.SearchInput.PromptStyle
	m.hctrl.pattern.TextStyle = m.BlurredStyle.SearchInput.TextStyle
	m.hctrl.pattern.BackgroundStyle = m.BlurredStyle.SearchInput.BackgroundStyle
	m.hctrl.pattern.PlaceholderStyle = m.BlurredStyle.SearchInput.PlaceholderStyle
	m.hctrl.pattern.CursorStyle = m.BlurredStyle.SearchInput.CursorStyle
}

// Init is part of the tea.Model interface.
func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) currentlySearching() bool {
	return m.hctrl.c.searching
}

func (m *Model) historyStartSearch() {
	m.hctrl.c.searching = true
	m.hctrl.c.prevPattern = ""
	m.hctrl.pattern.Prompt = m.SearchPrompt
	m.hctrl.pattern.Reset()
	m.hctrl.pattern.Focus()
	m.saveValue()
	m.resetNavCursor()
}

func (m *Model) resetNavCursor() {
	m.hctrl.c.cursor = len(m.history)
}

func (m *Model) cancelHistorySearch() (cmd tea.Cmd) {
	m.hctrl.c.searching = false
	m.hctrl.pattern.Blur()
	cmd = m.restoreValue()
	m.text.Focus()
	return cmd
}

func (m *Model) restoreValue() (cmd tea.Cmd) {
	cmd = m.updateValue(m.hctrl.c.prevValue, m.hctrl.c.prevCursor)
	m.hctrl.c.valueSaved = false
	m.hctrl.c.prevValue = ""
	m.hctrl.c.prevCursor = 0
	m.resetNavCursor()
	return cmd
}

func (m *Model) acceptSearch() {
	m.hctrl.c.searching = false
	m.hctrl.c.valueSaved = false
	m.hctrl.c.prevValue = ""
	m.hctrl.c.prevCursor = 0
	m.hctrl.pattern.Blur()
	m.text.Focus()
}

func (m *Model) incrementalSearch(nextMatch bool) (cmd tea.Cmd) {
	pat := m.hctrl.pattern.Value()
	if pat == m.hctrl.c.prevPattern {
		if !nextMatch {
			// Nothing changed, and no request for incremental search: do nothing.
			return
		}
		// Just nextMatch: continue incremental search below.
	} else {
		// Pattern changed, start again.
		m.resetNavCursor()
		m.hctrl.c.prevPattern = pat
	}

	i := m.hctrl.c.cursor - 1
	for ; i >= 0; i-- {
		entry := m.history[i]
		for j := len(entry) - len(pat); j >= 0; j-- {
			// TODO: use glob search instead of simple strings.
			if strings.HasPrefix(entry[j:], pat) {
				// It's a match!
				m.hctrl.pattern.Prompt = m.SearchPrompt
				m.hctrl.c.cursor = i
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

func (m *Model) updatePrompt() {
	prompt, nextPrompt := m.Prompt, m.NextPrompt
	if m.hidePrompt {
		prompt, nextPrompt = "", ""
	}
	promptWidth := max(rw.StringWidth(prompt), rw.StringWidth(nextPrompt))
	m.text.Prompt = ""
	m.text.SetPromptFunc(promptWidth, func(line int) string {
		if line == 0 {
			return prompt
		}
		return nextPrompt
	})
	// Recompute the width.
	m.text.SetWidth(m.maxWidth - 1)
	m.text.SetCursor(0)
}

func (m *Model) saveValue() {
	m.hctrl.c.valueSaved = true
	m.hctrl.c.prevValue = m.text.Value()
	m.hctrl.c.prevCursor = m.text.CursorPos()
}

func (m *Model) historyUp() (cmd tea.Cmd) {
	if !m.hctrl.c.valueSaved {
		m.saveValue()
	}
	if m.hctrl.c.cursor == 0 {
		return cmd
	}
	m.hctrl.c.cursor--
	entry := m.history[m.hctrl.c.cursor]
	return tea.Batch(cmd, m.updateValue(entry, len(entry)))
}

func (m *Model) historyDown() (cmd tea.Cmd) {
	if !m.hctrl.c.valueSaved {
		m.saveValue()
	}
	m.hctrl.c.cursor++
	if m.hctrl.c.cursor >= len(m.history) {
		return m.restoreValue()
	}
	entry := m.history[m.hctrl.c.cursor]
	return tea.Batch(cmd, m.updateValue(entry, len(entry)))
}

func (m *Model) autoComplete() (cmd tea.Cmd) {
	msgs, consume, extra := m.AutoComplete(m.text.ValueRunes(), m.text.Line(), m.text.CursorPos())
	if msgs != "" {
		cmd = tea.Batch(cmd, tea.Println(msgs))
	}
	if len(extra) == 0 {
		// No completions. do nothing.
		return cmd
	}

	// If there's more than 1 completion, show the list at the top.
	if len(extra) > 1 {
		var buf strings.Builder
		sp := ""
		lw := 0
		for _, e := range extra[1:] {
			if e != "" {
				ww := rw.StringWidth(e)
				if lw+ww >= m.maxWidth {
					sp = ""
					lw = 0
					buf.WriteByte('\n')
				}
				buf.WriteString(sp)
				buf.WriteString(e)
				sp = " "
				lw += ww + 1
			}
		}
		if buf.Len() > 0 {
			cmd = tea.Batch(cmd, tea.Println(buf.String()))
		}
	}

	// In any case, auto-complete the first item.
	if consume > 0 {
		m.text.DeleteCharactersBackward(consume)
	}
	m.text.InsertString(extra[0])

	if len(extra) == 1 {
		// If there was just 1 match, insert a space too.
		m.text.InsertRune(' ')
	}

	// Finally, refresh the size.
	cmd = tea.Batch(cmd, m.updateTextSz())
	return cmd
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

// Debug returns debug details about the state of the model.
func (m *Model) Debug() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "lastEvent: %+v\n", m.lastEvent)
	fmt.Fprintf(&buf, "history: %q\n", m.history)
	fmt.Fprintf(&buf, "maxHeight: %d, maxWidth: %d\n", m.maxHeight, m.maxWidth)
	fmt.Fprintf(&buf, "hidePrompt: %v\n", m.hidePrompt)
	fmt.Fprintf(&buf, "hctrl.c: %+v\n", m.hctrl.c)
	fmt.Fprintf(&buf, "htctrl.pattern: %q\n", m.hctrl.pattern.Value())
	return buf.String()
}

// SetWidth changes the width of the editor.
func (m *Model) SetWidth(w int) {
	w = clamp(w, 1, m.maxWidth)
	m.text.SetWidth(w - 1)
	m.hctrl.pattern.Width = w - 1
	m.help.Width = w - 1
}

// Update is the Bubble Tea event handler.
// This is part of the tea.Model interface.
func (m *Model) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	stop := false

	m.lastEvent = imsg
	switch msg := imsg.(type) {
	case tea.WindowSizeMsg:
		m.maxWidth = msg.Width
		m.maxHeight = msg.Height
		m.SetWidth(msg.Width - 1)
		cmd = tea.Batch(cmd, m.updateTextSz())

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Debug):
			m.debugMode = !m.debugMode

		case key.Matches(msg, m.KeyMap.HideShowPrompt):
			m.hidePrompt = !m.hidePrompt
			m.updatePrompt()
			cmd = tea.Batch(cmd, m.updateTextSz())
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.AutoComplete):
			if m.AutoComplete == nil {
				// Pass-through to the editor.
				break
			}
			if m.currentlySearching() {
				m.acceptSearch()
			}
			cmd = m.autoComplete()
			imsg = 0 // consume message

		case key.Matches(msg, m.KeyMap.SignalQuit):
			return m, tea.Exec(doProgram(func() {
				pr, err := os.FindProcess(os.Getpid())
				if err != nil {
					// No-op.
					return
				}
				pr.Signal(syscall.SIGQUIT)
			}), nil)

		case key.Matches(msg, m.KeyMap.SignalTTYStop):
			return m, tea.Exec(doProgram(func() {
				pr, err := os.FindProcess(os.Getpid())
				if err != nil {
					// No-op.
					return
				}
				pr.Signal(syscall.SIGTSTP)
			}), nil)

		case key.Matches(msg, m.KeyMap.Refresh):
			return m, tea.Exec(doProgram(termenv.ClearScreen), nil)

		case key.Matches(msg, m.KeyMap.EndOfInput):
			if m.text.AtBeginningOfLine() {
				if m.currentlySearching() {
					cmd = tea.Batch(cmd, m.cancelHistorySearch())
				}
				m.Err = io.EOF
				stop = true
				imsg = nil // consume message
			} else if m.DeleteCharIfNotEOF {
				m.text.DeleteCharacterForward()
				imsg = nil // consume message
			}

		case key.Matches(msg, m.KeyMap.AbortSearch):
			if m.currentlySearching() {
				cmd = tea.Batch(cmd, m.cancelHistorySearch())
				imsg = nil // consume message
			}

		case key.Matches(msg, m.KeyMap.SearchBackward):
			if m.currentlySearching() {
				m.incrementalSearch(true /* nextMatch */)
			} else {
				// Start completion.
				m.historyStartSearch()
			}
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.HistoryPrevious):
			if m.currentlySearching() {
				m.acceptSearch()
			}
			m.historyUp()
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.HistoryNext):
			if m.currentlySearching() {
				m.acceptSearch()
			}
			m.historyDown()
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.Interrupt):
			imsg = nil // consume message
			if m.text.EmptyValue() {
				m.Err = ErrInterrupted
				stop = true
				break
			}
			if m.currentlySearching() {
				// Stop the completion. Do nothing further.
				cmd = tea.Batch(cmd, m.cancelHistorySearch())
			} else {
				m.text.SetValue("")
			}

		case key.Matches(msg, m.KeyMap.AlwaysNewline):
			if m.currentlySearching() {
				// Stop the completion first.
				m.acceptSearch()
			}
			m.text.InsertNewline()
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.InsertNewline):
			if m.currentlySearching() {
				// Stop the completion first.
				m.acceptSearch()
			}
			if m.CheckInputComplete == nil ||
				m.CheckInputComplete(m.text.ValueRunes(), m.text.Line(), m.text.CursorPos()) {
				stop = true
				// Fallthrough: we want the enter key to be processed by the
				// textarea so that there's a final empty line in the display.
			}

		case key.Matches(msg, m.KeyMap.LinePrevious):
			if m.currentlySearching() {
				m.acceptSearch()
			}
			if m.text.AtFirstLineOfInputAndView() {
				m.historyUp()
				imsg = nil // consume message
			}

		case key.Matches(msg, m.KeyMap.LineNext):
			if m.currentlySearching() {
				m.acceptSearch()
			}
			if m.text.AtLastLineOfInputAndView() {
				m.historyDown()
				imsg = nil // consume message
			}

		case key.Matches(msg, m.KeyMap.CharacterForward) ||
			key.Matches(msg, m.KeyMap.CharacterBackward) ||
			key.Matches(msg, m.KeyMap.WordForward) ||
			key.Matches(msg, m.KeyMap.WordBackward):
			if m.currentlySearching() {
				m.acceptSearch()
			}

		}
	}

	if m.currentlySearching() {
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
	m.hidePrompt = false
	m.debugMode = false
	m.hctrl.c.valueSaved = false
	m.hctrl.c.prevValue = ""
	m.hctrl.c.prevCursor = 0
	m.text.CharLimit = m.CharLimit
	// Width will be set by Update below on init.
	m.text.SetHeight(1)
	m.text.Reset()
	m.Focus()
}

// View renders the text area in its current state.
// This is part of the tea.Model interface.
func (m Model) View() string {
	var buf strings.Builder
	if m.debugMode {
		fmt.Fprintf(&buf, "editline:\n%s\ntextarea:\n%s\n", m.Debug(), m.text.Debug())
	}

	buf.WriteString(m.text.View())
	if m.currentlySearching() {
		buf.WriteByte('\n')
		buf.WriteString(m.hctrl.pattern.View())
	}
	return buf.String()
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
