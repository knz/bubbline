package complete

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	rw "github.com/mattn/go-runewidth"
)

// Values is the type of the possible completion inputs.
type Values struct {
	// Prefill is the string to insert at point, even when there are no
	// completions.
	Prefill string

	// Categories is the list of completion categories.
	Categories []string

	// Completions is the list of completions per category.
	Completions map[string][]string
}

// Styles contain style definitions for the completions component.
type Styles struct {
	FocusedTitleBar             lipgloss.Style
	FocusedTitle                lipgloss.Style
	BlurredTitleBar             lipgloss.Style
	BlurredTitle                lipgloss.Style
	Item                        lipgloss.Style
	SelectedItem                lipgloss.Style
	Spinner                     lipgloss.Style
	FilterPrompt                lipgloss.Style
	FilterCursor                lipgloss.Style
	PaginationStyle             lipgloss.Style
	DefaultFilterCharacterMatch lipgloss.Style
	ActivePaginationDot         lipgloss.Style
	InactivePaginationDot       lipgloss.Style
	ArabicPagination            lipgloss.Style
	DividerDot                  lipgloss.Style
}

// DefaultStyles returns a set of default style definitions for the
// completions component.
var DefaultStyles = func() (c Styles) {
	ls := list.DefaultStyles()
	subtle := lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}

	c.Item = lipgloss.NewStyle().PaddingLeft(1)
	c.SelectedItem = lipgloss.NewStyle().PaddingLeft(1).Foreground(lipgloss.Color("170"))

	c.FocusedTitleBar = lipgloss.NewStyle()
	c.BlurredTitleBar = lipgloss.NewStyle()
	c.FocusedTitle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	c.BlurredTitle = c.FocusedTitle.Copy().Foreground(subtle)
	c.Spinner = ls.Spinner
	c.FilterPrompt = ls.FilterPrompt
	c.FilterCursor = ls.FilterCursor
	c.PaginationStyle = lipgloss.NewStyle()
	c.DefaultFilterCharacterMatch = ls.DefaultFilterCharacterMatch
	c.ActivePaginationDot = ls.ActivePaginationDot
	c.InactivePaginationDot = ls.InactivePaginationDot
	c.ArabicPagination = ls.ArabicPagination
	c.DividerDot = ls.DividerDot
	return c
}()

// KeyMap defines keybindings for navigating the completions.
type KeyMap struct {
	list.KeyMap
	NextCompletions  key.Binding
	PrevCompletions  key.Binding
	AcceptCompletion key.Binding
	Abort            key.Binding
}

// DefaultKeyMap is the default set of key bindings.
var DefaultKeyMap = KeyMap{
	KeyMap: list.KeyMap{
		CursorUp:             key.NewBinding(key.WithKeys("up", "ctrl+p"), key.WithHelp("C-p/↑", "prev entry")),
		CursorDown:           key.NewBinding(key.WithKeys("down", "ctrl+n"), key.WithHelp("C-n/↓", "next entry")),
		NextPage:             key.NewBinding(key.WithKeys("right", "pgdown"), key.WithHelp("←/pgdown", "prev page/column")),
		PrevPage:             key.NewBinding(key.WithKeys("left", "pgup"), key.WithHelp("→/pgup", "next page/column")),
		GoToStart:            key.NewBinding(key.WithKeys("ctrl+a", "home"), key.WithHelp("C-a/home", "start of column")),
		GoToEnd:              key.NewBinding(key.WithKeys("ctrl+e", "end"), key.WithHelp("C-e/end", "end of column")),
		Filter:               key.NewBinding(key.WithKeys("/", ""), key.WithHelp("/", "filter")),
		ClearFilter:          key.NewBinding(key.WithKeys("ctrl+g"), key.WithHelp("C-g", "clear/cancel")),
		CancelWhileFiltering: key.NewBinding(key.WithKeys("ctrl+g"), key.WithHelp("C-g", "clear/cancel")),
		AcceptWhileFiltering: key.NewBinding(key.WithKeys("enter", "ctrl+j"), key.WithHelp("C-j/enter", "accept filter")),
		ShowFullHelp:         key.NewBinding(key.WithKeys("alt+?"), key.WithHelp("M-?", "toggle key help")),
		CloseFullHelp:        key.NewBinding(key.WithKeys("alt+?"), key.WithHelp("M-?", "toggle key help")),
	},
	NextCompletions:  key.NewBinding(key.WithKeys("tab", "alt+n"), key.WithHelp("M-n/tab", "next column")),
	PrevCompletions:  key.NewBinding(key.WithKeys("alt+p"), key.WithHelp("M-p/tab", "prev column")),
	AcceptCompletion: key.NewBinding(key.WithKeys("enter", "ctrl+j"), key.WithHelp("C-j/enter", "accept")),
	Abort:            key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("C-c", "close/cancel")),
}

// Model is the model that implements the completion
// selector widget.
type Model struct {
	Err error

	// KeyMap is the key bindings for navigating the completions.
	KeyMap KeyMap

	// Styles is the styles to use for display.
	Styles Styles

	// AcceptedValue is the result of the selection.
	AcceptedValue string

	width     int
	height    int
	maxHeight int
	focused   bool

	values Values

	selectedList int
	listItems    [][]list.Item
	valueLists   []*list.Model
}

func (m *Model) Debug() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "width: %d, height: %d, maxHeight: %d\n", m.width, m.height, m.maxHeight)
	fmt.Fprintf(&buf, "num lists: %d\n", len(m.valueLists))
	fmt.Fprintf(&buf, "selectedList: %d\n", m.selectedList)
	if len(m.valueLists) > 0 {
		fmt.Fprintf(&buf, "selected item: %v\n", m.valueLists[m.selectedList].SelectedItem())
	}
	fmt.Fprintf(&buf, "accepted: %q / err %v\n", m.AcceptedValue, m.Err)
	return buf.String()
}

var _ tea.Model = (*Model)(nil)

func New() Model {
	return Model{
		KeyMap:  DefaultKeyMap,
		Styles:  DefaultStyles,
		focused: true,
	}
}

type stringItem string

var _ list.Item = stringItem("")

// FilterValue implements the list.Item interface
func (s stringItem) FilterValue() string { return string(s) }

func convertToItems(items []string) (res []list.Item, maxWidth int) {
	res = make([]list.Item, len(items))
	for i, it := range items {
		// TODO(knz): Support multi-line items.
		maxWidth = max(maxWidth, rw.StringWidth(it))
		res[i] = stringItem(it)
	}
	return res, maxWidth
}

type renderer struct {
	m       *Model
	listIdx int
	width   int
}

var _ list.ItemDelegate = (*renderer)(nil)

// Render is part of the list.ItemDelegate interface.
func (r *renderer) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(stringItem)
	if !ok {
		return
	}
	s := string(i)
	iw := rw.StringWidth(s)
	if iw < r.width {
		s += strings.Repeat(" ", r.width-iw)
	}
	st := &r.m.Styles
	fn := st.Item.Render
	if r.m.selectedList == r.listIdx && index == m.Index() {
		fn = st.SelectedItem.Render
	}
	fmt.Fprint(w, fn(s))
}

// Height is part of the list.ItemDelegae interface.
func (r *renderer) Height() int {
	// TODO(knz): Support multi-line items, e.g. identifiers
	// containing a newline character.
	return 1
}

// Spacing is part of the list.ItemDelegate interface.
func (r *renderer) Spacing() int { return 0 }

// Update is part of the list.ItemDelegate interface.
func (r *renderer) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// SetWidth changes the width.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetHeight changes the height.
func (m *Model) SetHeight(height int) {
	m.height = clamp(height, 1, m.maxHeight)
	for _, l := range m.valueLists {
		l.SetHeight(m.height)
		// Force recomputing the keybindigns, which
		// is dependent on the page size.
		l.SetFilteringEnabled(true)
	}
}

// GetHeight retrieves the current height.
func (m *Model) GetHeight() int {
	return m.height
}

// GetHeight retrieves the maximum height.
func (m *Model) GetMaxHeight() int {
	return m.maxHeight
}

// SetValues resets the values. It also recomputes the height.
func (m *Model) SetValues(values Values) {
	m.Err = nil
	m.selectedList = 0
	m.values = values
	m.valueLists = make([]*list.Model, len(values.Categories))
	m.listItems = make([][]list.Item, len(values.Categories))
	const stdHeight = 10
	listDecorationRows :=
		1 +
			max(
				m.Styles.FocusedTitleBar.GetVerticalPadding(),
				m.Styles.BlurredTitleBar.GetVerticalPadding()) +
			max(
				m.Styles.FocusedTitleBar.GetVerticalMargins(),
				m.Styles.BlurredTitleBar.GetVerticalMargins()) +
			1 +
			m.Styles.PaginationStyle.GetVerticalPadding() +
			// (facepalm) the list widget forces a vertical margin of 1...
			max(1, m.Styles.PaginationStyle.GetVerticalMargins())
	m.maxHeight = listDecorationRows

	perItemHeight := 1 + max(
		m.Styles.Item.GetVerticalPadding(),
		m.Styles.SelectedItem.GetVerticalPadding())

	for i, category := range values.Categories {
		var maxWidth int
		m.listItems[i], maxWidth = convertToItems(values.Completions[category])
		m.maxHeight = max(m.maxHeight, len(m.listItems[i])*perItemHeight+listDecorationRows)
		maxWidth = max(maxWidth+1, rw.StringWidth(category))
		r := &renderer{m: m, listIdx: i, width: maxWidth}
		l := list.New(m.listItems[i], r, maxWidth, stdHeight)
		l.Title = category
		l.KeyMap = m.KeyMap.KeyMap
		l.DisableQuitKeybindings()
		l.SetShowHelp(false)
		l.SetShowStatusBar(false)
		m.valueLists[i] = &l
	}

	// Propagate the logical weights to all lists.
	m.SetHeight(m.maxHeight)

	wasFocused := m.focused
	m.Blur()
	if wasFocused {
		m.Focus()
	}
}

// MatchesKeys returns true when the completion
// editor can use the given key message.
func (m *Model) MatchesKey(msg tea.KeyMsg) bool {
	if m.focused == false || len(m.valueLists) == 0 {
		return false
	}
	curList := m.valueLists[m.selectedList]
	switch {
	case key.Matches(msg,
		m.KeyMap.CursorUp,
		m.KeyMap.CursorDown,
		m.KeyMap.GoToStart,
		m.KeyMap.GoToEnd,
		m.KeyMap.Filter,
		m.KeyMap.ClearFilter,
		m.KeyMap.CancelWhileFiltering,
		m.KeyMap.AcceptWhileFiltering,
		m.KeyMap.PrevCompletions,
		m.KeyMap.NextCompletions,
		m.KeyMap.NextPage,
		m.KeyMap.PrevPage,
		m.KeyMap.Abort):
		return true
	case !curList.SettingFilter() &&
		key.Matches(msg, m.KeyMap.AcceptCompletion):
		return true
	case curList.SettingFilter():
		return true
	}
	return false
}

// Focus places the focus on the completion editor.
func (m *Model) Focus() {
	m.focused = true
	if len(m.valueLists) == 0 {
		return
	}
	l := m.valueLists[m.selectedList]
	l.Styles.Title = m.Styles.FocusedTitle
	l.Styles.TitleBar = m.Styles.FocusedTitleBar
}

// Blur removes the focus from the completion editor.
func (m *Model) Blur() {
	m.focused = false
	for _, l := range m.valueLists {
		l.Styles.Title = m.Styles.BlurredTitle
		l.Styles.TitleBar = m.Styles.BlurredTitleBar
	}
}

func (m *Model) prevCompletions() {
	wasFocused := m.focused
	m.Blur()
	m.selectedList = (m.selectedList + len(m.valueLists) - 1) % len(m.valueLists)
	curList := m.valueLists[m.selectedList]
	curList.Select(len(curList.VisibleItems()) - 1)
	if wasFocused {
		m.Focus()
	}
}

func (m *Model) nextCompletions() {
	wasFocused := m.focused
	m.Blur()
	m.selectedList = (m.selectedList + 1) % len(m.valueLists)
	m.valueLists[m.selectedList].Select(0)
	if wasFocused {
		m.Focus()
	}
}

// Init implements the tea.Model interface.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface.
func (m *Model) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.valueLists) == 0 {
		m.Err = io.EOF
		return m, nil
	}

	curList := m.valueLists[m.selectedList]
	switch msg := imsg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Abort):
			m.AcceptedValue = ""
			m.Err = io.EOF
			imsg = nil
		case !curList.SettingFilter():
			switch {
			case key.Matches(msg, m.KeyMap.PrevCompletions):
				m.prevCompletions()
				imsg = nil
			case key.Matches(msg, m.KeyMap.NextCompletions):
				m.nextCompletions()
				imsg = nil
			case key.Matches(msg, m.KeyMap.NextPage):
				if curList.Paginator.Page >= curList.Paginator.TotalPages-1 {
					m.nextCompletions()
					imsg = nil
				}
			case key.Matches(msg, m.KeyMap.PrevPage):
				if curList.Paginator.Page == 0 {
					m.prevCompletions()
					imsg = nil
				}
			case key.Matches(msg, m.KeyMap.AcceptCompletion):
				v := curList.SelectedItem().(stringItem)
				m.AcceptedValue = string(v)
				m.Err = io.EOF
				imsg = nil
			}
		}
	}
	if imsg == nil {
		return m, nil
	}
	newModel, cmd := m.valueLists[m.selectedList].Update(imsg)
	// By default, the list blocks the enter key when the
	// filtering prompt is open but there is no filter entered.
	// We don't like this - enter should just accept the current item.
	newModel.KeyMap.AcceptWhileFiltering.SetEnabled(true)
	m.valueLists[m.selectedList] = &newModel
	return m, cmd
}

// View implements the tea.Model interface.
func (m *Model) View() string {
	contents := make([]string, len(m.valueLists))
	for i, l := range m.valueLists {
		contents[i] = l.View()
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, contents...)
}

// ShortHelp is part of the help.KeyMap interface.
func (m *Model) ShortHelp() []key.Binding {
	if len(m.valueLists) == 0 {
		return nil
	}

	kb := []key.Binding{
		m.KeyMap.Abort,
	}

	curList := m.valueLists[m.selectedList]
	if !curList.SettingFilter() {
		kb = append(kb,
			m.KeyMap.NextCompletions,
			m.KeyMap.AcceptCompletion,
		)
	}
	return append(kb, curList.ShortHelp()...)
}

// FullHelp is part of the help.KeyMap interface.
func (m *Model) FullHelp() [][]key.Binding {
	if len(m.valueLists) == 0 {
		return nil
	}
	curList := m.valueLists[m.selectedList]
	kb := [][]key.Binding{{
		m.KeyMap.NextCompletions,
		m.KeyMap.PrevCompletions,
		m.KeyMap.AcceptCompletion,
		m.KeyMap.Abort,
	}}
	kb = append(kb, curList.FullHelp()...)
	return kb
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}
