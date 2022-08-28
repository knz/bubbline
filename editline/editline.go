package editline

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/editline/internal/textarea"
	rw "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
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
	AlwaysComplete  key.Binding
	MoreHelp        key.Binding
	ReflowLine      key.Binding
	ReflowAll       key.Binding
	MoveToBegin     key.Binding
	MoveToEnd       key.Binding
}

// DefaultKeyMap is the default set of key bindings.
var DefaultKeyMap = KeyMap{
	KeyMap: textarea.DefaultKeyMap,

	AlwaysNewline:   key.NewBinding(key.WithKeys("ctrl+j"), key.WithHelp("C-j", "force newline")),
	AlwaysComplete:  key.NewBinding(key.WithKeys("alt+enter", "alt+\r"), key.WithHelp("M-⤶/M-C-m", "force complete")),
	AutoComplete:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "try autocomplete")),
	Interrupt:       key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("C-c", "clear/cancel")),
	SignalQuit:      key.NewBinding(key.WithKeys(`ctrl+\`)),
	SignalTTYStop:   key.NewBinding(key.WithKeys("ctrl+z")),
	Refresh:         key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("C-l", "refresh display")),
	EndOfInput:      key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("C-d", "erase/stop")),
	AbortSearch:     key.NewBinding(key.WithKeys("ctrl+g"), key.WithDisabled()),
	SearchBackward:  key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("C-r", "search hist"), key.WithDisabled()),
	HistoryPrevious: key.NewBinding(key.WithKeys("alt+p"), key.WithHelp("M-p", "prev history entry"), key.WithDisabled()),
	HistoryNext:     key.NewBinding(key.WithKeys("alt+n"), key.WithHelp("M-n", "next history entry"), key.WithDisabled()),
	HideShowPrompt:  key.NewBinding(key.WithKeys("alt+."), key.WithHelp("M-.", "hide/show prompt")),
	MoreHelp:        key.NewBinding(key.WithKeys("alt+?"), key.WithHelp("M-?", "toggle key help")),
	ReflowLine:      key.NewBinding(key.WithKeys("alt+q"), key.WithHelp("M-q", "reflow line")),
	ReflowAll:       key.NewBinding(key.WithKeys("alt+Q"), key.WithHelp("M-S-q", "reflow all")),
	Debug:           key.NewBinding(key.WithKeys("ctrl+_", "ctrl+@"), key.WithHelp("C-_/C-@", "debug mode"), key.WithDisabled()),
	MoveToBegin:     key.NewBinding(key.WithKeys("alt+<", "ctrl+home"), key.WithHelp("M-</C-home", "go to begin")),
	MoveToEnd:       key.NewBinding(key.WithKeys("alt+>", "ctrl+end"), key.WithHelp("M->/C-end", "go to end")),
}

// AutoCompleteFn is called upon the user pressing the
// autocomplete key. The callback is provided the text of the input
// and the position of the cursor in the input.
// The returned msg is printed above the input box.
//
// If the moveRight is non-zero, the cursor is moved that number to the right.
// If the deleteLeft are non-zero, that number of characters
// is erased before the cursor, after it has been moved.
// Then the first string in the returned extraInput value is added at the cursor position.
// If there is more than 1 entry in the returned extraInput, they are
// displayed above the input box too.
type AutoCompleteFn func(entireInput [][]rune, line, col int) (msg string, moveRight, deleteLeft int, extraInput complete.Values)

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

	// AutoComplete is the AutoCompleteFn to use.
	AutoComplete AutoCompleteFn

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

	// Reflow, if defined, is used for the reflowing commands (M-q/M-Q).
	// The info returned value, if any, is displayed as an informational
	// message above the editor.
	Reflow func(all bool, currentText string, targetWidth int) (changed bool, newText, info string)

	// SearchPrompt is the prompt displayed before the history search pattern.
	SearchPrompt string
	// SearchPromptNotFound is the prompt displayed before the history search pattern,
	// when no match is found.
	SearchPromptNotFound string

	// ShowLineNumbers if true shows line numbers at the beginning
	// of each input line.
	// Only takes effect at Reset() or Focus().
	ShowLineNumbers bool

	showCompletions        bool
	consumeAfterCompletion int
	completions            complete.Model

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

	help help.Model

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
	m := &Model{
		text:                 textarea.New(),
		Err:                  nil,
		KeyMap:               DefaultKeyMap,
		MaxHistorySize:       0, // no limit
		Reflow:               DefaultReflow,
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
		help:                 help.New(),
		completions:          complete.New(),
	}
	m.text.KeyMap.Paste.Unbind() // paste from clipboard not supported.
	m.hctrl.pattern = textinput.New()
	m.hctrl.pattern.Placeholder = "enter search term, or C-g to cancel search"
	m.Reset()
	return m
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
	m.checkHistoryEnabled()
	m.resetNavCursor()
}

// SetDebugEnabled enables/disables the debug mode binding.
// When disabling it, it also proactively disables debugging if currently enabled.
func (m *Model) SetDebugEnabled(enable bool) {
	m.KeyMap.Debug.SetEnabled(enable)
	if !enable {
		m.debugMode = false
	}
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
	m.checkHistoryEnabled()
	m.resetNavCursor()
}

func (m *Model) checkHistoryEnabled() {
	enabled := len(m.history) > 0
	m.KeyMap.AbortSearch.SetEnabled(enabled)
	m.KeyMap.SearchBackward.SetEnabled(enabled)
	m.KeyMap.HistoryPrevious.SetEnabled(enabled)
	m.KeyMap.HistoryNext.SetEnabled(enabled)
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
	m.completions.Focus()

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
	m.completions.Blur()
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
	textHeight := m.text.LogicalHeight()

	remaining := m.maxHeight - 1
	if m.showCompletions {
		// Don't let the completions exceed half of the screen size.
		ch := m.completions.GetMaxHeight()
		if ch+textHeight > remaining {
			const minCompletionHeight = 4 // 1 row title, 1 row entry, 2 rows pagination
			newCompletionHeight := clamp(ch, minCompletionHeight, remaining-textHeight)
			m.completions.SetHeight(newCompletionHeight)
		}
		remaining -= m.completions.GetHeight()
	}

	newHeight := clamp(m.text.LogicalHeight(), 1, remaining)

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
	msgs, moveRight, deleteLeft, extra := m.AutoComplete(m.text.ValueRunes(), m.text.Line(), m.text.CursorPos())
	if msgs != "" {
		// TODO(knz): maybe display the help using a viewport widget?
		cmd = tea.Batch(cmd, tea.Println(msgs))
	}
	if len(extra.Prefill) == 0 && len(extra.Completions) == 0 {
		// No completions. do nothing.
		return cmd
	}

	// Move the cursor to the right if requested.
	if moveRight > 0 {
		m.text.CursorRight(moveRight)
	}

	// In any case, auto-complete the prefill text.
	if len(extra.Prefill) > 0 {
		if deleteLeft > 0 {
			m.text.DeleteCharactersBackward(deleteLeft)
		}
		m.text.InsertString(extra.Prefill)
		deleteLeft = rw.StringWidth(extra.Prefill)
	}

	if len(extra.Completions) == 0 {
		// If there was just a prefill, insert a space. We're done
		// with auto-completion.
		m.text.InsertRune(' ')
	} else {
		m.showCompletions = true
		m.consumeAfterCompletion = deleteLeft
		m.completions.SetValues(extra)
		m.completions.Focus()
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
	fmt.Fprintf(&buf, "showComp: %v, consume: %d\n", m.showCompletions, m.consumeAfterCompletion)
	fmt.Fprintf(&buf, "htctrl.pattern: %q\n", m.hctrl.pattern.Value())
	return buf.String()
}

// SetWidth changes the width of the editor.
func (m *Model) SetWidth(w int) {
	w = clamp(w, 1, m.maxWidth)
	m.text.SetWidth(w - 1)
	m.completions.SetWidth(w - 1)
	m.hctrl.pattern.Width = w - 1
	m.help.Width = w - 1
}

// DefaultReflow is the default/initial value of Reflow.
func DefaultReflow(
	allText bool, currentText string, targetWidth int,
) (changed bool, newText, info string) {
	if rw.StringWidth(currentText) <= targetWidth {
		return false, currentText, ""
	}
	return true, wordwrap.String(currentText, targetWidth), ""
}

// reflowLine reflows the current line.
func (m *Model) reflowLine() (cmd tea.Cmd) {
	if m.Reflow == nil {
		return nil
	}
	s := m.text.CurrentLine()
	changed, newText, info := m.Reflow(false /*all*/, s, m.text.Width()-1)
	if !changed {
		return nil
	}
	m.text.ClearLine()
	m.text.InsertString(newText)
	if info != "" {
		cmd = tea.Println(info)
	}
	return tea.Batch(cmd, m.updateTextSz())
}

// reflowAll reflows the entire text.
func (m *Model) reflowAll() (cmd tea.Cmd) {
	if m.Reflow == nil {
		return nil
	}
	s := m.text.Value()
	changed, newText, info := m.Reflow(true /*all*/, s, m.text.Width()-1)
	if !changed {
		return nil
	}
	m.text.SetValue(newText)
	if info != "" {
		cmd = tea.Println(info)
	}
	return tea.Batch(cmd, m.updateTextSz())
}

// handleCompletions navigates through the completion screen.
func (m *Model) handleCompletions(imsg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.completions.Update(imsg)
	if m.completions.Err == nil {
		return m, cmd
	}
	v := m.completions.AcceptedValue
	if v != "" {
		m.text.DeleteCharactersBackward(m.consumeAfterCompletion)
		m.text.InsertString(v)
		m.text.InsertRune(' ')
	}
	m.showCompletions = false
	m.completions.Blur()
	return m, tea.Batch(cmd, m.updateTextSz())
}

// Update is the Bubble Tea event handler.
// This is part of the tea.Model interface.
func (m *Model) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	m.lastEvent = imsg

	switch msg := imsg.(type) {
	case tea.WindowSizeMsg:
		m.maxWidth = msg.Width
		m.maxHeight = msg.Height
		m.SetWidth(msg.Width - 1)
		return m, m.updateTextSz()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Debug):
			m.debugMode = !m.debugMode

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

		case key.Matches(msg, m.KeyMap.MoreHelp):
			m.help.ShowAll = !m.help.ShowAll
			imsg = nil // consume message

		default:
			m.help.ShowAll = false

			if m.showCompletions && !m.completions.MatchesKey(msg) {
				// Currently displaying completions, but the widget
				// is not accepting this keystroke. Cancel completions
				// altogether and simply keep the input.
				m.showCompletions = false
				m.completions.Blur()
			}
		}
	}

	if m.showCompletions {
		return m.handleCompletions(imsg)
	}

	var cmd tea.Cmd
	stop := false

	switch msg := imsg.(type) {
	case tea.KeyMsg:
		switch {
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

		case key.Matches(msg, m.KeyMap.AlwaysComplete):
			if m.currentlySearching() {
				// Stop the completion first.
				m.acceptSearch()
			}
			stop = true
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.MoveToBegin):
			if m.currentlySearching() {
				// Stop the completion first.
				m.acceptSearch()
			}
			m.text.MoveToBegin()
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.MoveToEnd):
			if m.currentlySearching() {
				// Stop the completion first.
				m.acceptSearch()
			}
			m.text.MoveToEnd()
			imsg = nil // consume message

		case key.Matches(msg, m.KeyMap.InsertNewline):
			if m.currentlySearching() {
				// Stop the completion first.
				m.acceptSearch()
				// If we were searching, do not process the enter key -- otherwise
				// it will add a newline character in the middle of the completion.
				imsg = nil // consume message
			}
			if m.CheckInputComplete == nil ||
				m.CheckInputComplete(m.text.ValueRunes(), m.text.Line(), m.text.CursorPos()) {
				stop = true

				// Avoid processing the enter key, for otherwise it may insert
				// an excess newline in the middle of the input.
				imsg = nil // consume message
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

		case key.Matches(msg, m.KeyMap.ReflowLine):
			if m.currentlySearching() {
				m.acceptSearch()
			}
			cmd = tea.Batch(cmd, m.reflowLine())
			imsg = nil

		case key.Matches(msg, m.KeyMap.ReflowAll):
			if m.currentlySearching() {
				m.acceptSearch()
			}
			cmd = tea.Batch(cmd, m.reflowAll())
			imsg = nil

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
		m.help.ShowAll = false
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
	m.showCompletions = false
	m.completions.Blur()
	m.hctrl.c.valueSaved = false
	m.hctrl.c.prevValue = ""
	m.hctrl.c.prevCursor = 0
	m.text.CharLimit = m.CharLimit
	// Width will be set by Update below on init.
	m.text.SetHeight(1)
	m.completions.SetHeight(1)
	m.text.Reset()
	m.Focus()
}

// View renders the text area in its current state.
// This is part of the tea.Model interface.
func (m Model) View() string {
	var buf strings.Builder
	if m.debugMode {
		buf.WriteString(
			lipgloss.JoinHorizontal(lipgloss.Top,
				fmt.Sprintf("editline:\n%s", wordwrap.String(m.Debug(), 50)),
				" ",
				fmt.Sprintf("textarea:\n%s", wordwrap.String(m.text.Debug(), 50)),
				" ",
				fmt.Sprintf("comp:\n%s", wordwrap.String(m.completions.Debug(), 50))))
		buf.WriteByte('\n')
	}

	if m.showCompletions {
		buf.WriteString(m.completions.View())
		buf.WriteByte('\n')
	}
	buf.WriteString(m.text.View())
	if m.currentlySearching() {
		buf.WriteByte('\n')
		buf.WriteString(m.hctrl.pattern.View())
	} else {
		buf.WriteByte('\n')
		buf.WriteString(m.help.View(m))
	}
	return buf.String()
}

// ShortHelp is part of the help.KeyMap interface.
func (m Model) ShortHelp() []key.Binding {
	k := m.KeyMap
	kb := []key.Binding{
		k.MoreHelp,
	}
	if m.showCompletions {
		return append(kb, m.completions.ShortHelp()...)
	}
	return append(kb,
		k.EndOfInput, k.Interrupt, k.SearchBackward, k.HideShowPrompt,
	)
}

// FullHelp is part of the help.KeyMap interface.
func (m Model) FullHelp() [][]key.Binding {
	if m.showCompletions {
		return m.completions.FullHelp()
	}
	k := m.KeyMap
	return [][]key.Binding{
		{
			k.MoreHelp,
			k.CharacterForward,
			k.WordForward,
			k.MoveToEnd,
			key.NewBinding(key.WithKeys("_"), key.WithHelp("del", "del next char")), // k.DeleteCharacterForward,
			k.DeleteWordForward,
			k.LineEnd,
			k.DeleteAfterCursor,
			k.LineNext,
			k.HistoryNext,
			k.ReflowLine,
		},
		{
			k.Interrupt,
			k.CharacterBackward,
			k.WordBackward,
			k.MoveToBegin,
			k.DeleteCharacterBackward,
			k.DeleteWordBackward,
			k.LineStart,
			k.DeleteBeforeCursor,
			k.LinePrevious,
			k.HistoryPrevious,
			k.ReflowAll,
		},
		{
			k.HideShowPrompt,
			key.NewBinding(key.WithKeys("_"), key.WithHelp("C-d", "del next char/EOF")),
			k.AlwaysNewline,
			k.AlwaysComplete,
			k.Refresh,
			k.ToggleOverwriteMode,
			k.TransposeCharacterBackward,
			k.LowercaseWordForward,
			k.UppercaseWordForward,
			k.SearchBackward,
			k.AutoComplete,
		},
	}
	return nil
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
