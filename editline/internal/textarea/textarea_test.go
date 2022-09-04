package textarea

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
)

func TestTextArea(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		m := testModel{
			text: New(),
		}

		catwalk.RunModel(t, path, &m,
			catwalk.WithUpdater(testCmd),
			catwalk.WithObserver("value", func(out io.Writer, m tea.Model) error {
				s := m.(*testModel).text.Value()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("runes", func(out io.Writer, m tea.Model) error {
				s := m.(*testModel).text.ValueRunes()
				fmt.Fprintf(out, "%+v", s)
				return nil
			}),
			catwalk.WithObserver("curline", func(out io.Writer, m tea.Model) error {
				s := m.(*testModel).text.CurrentLine()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("props", func(out io.Writer, m tea.Model) error {
				t := &m.(*testModel).text
				fmt.Fprintf(out, "Focused: %v\n", t.Focused())
				fmt.Fprintf(out, "Width: %d, Height: %d, LogicalHeight: %d\n", t.Width(), t.Height(), t.LogicalHeight())
				fmt.Fprintf(out, "Length: %d, LineCount: %d, NumLinesInValue: %d, EmptyValue: %v\n",
					t.Length(), t.LineCount(), t.NumLinesInValue(), t.EmptyValue(),
				)
				return nil
			}),
			catwalk.WithObserver("pos", func(out io.Writer, m tea.Model) error {
				t := &m.(*testModel).text
				fmt.Fprintf(out, "Line: %d, Pos %d, (row %d, col %d, lastCharOffset %d)\n",
					t.Line(), t.CursorPos(), t.row, t.col, t.lastCharOffset,
				)
				fmt.Fprintf(out, "LineInfo: %+v\n", t.LineInfo())
				fmt.Fprintf(out, "AtBeginningOfLine: %v, AtFirstLineOfInputAndView: %v, AtLastLineOfInputAndView: %v\n",
					t.AtBeginningOfLine(), t.AtFirstLineOfInputAndView(), t.AtLastLineOfInputAndView(),
				)
				return nil
			}),
		)
	})
}

type testModel struct {
	text Model
}

func (t *testModel) Init() tea.Cmd { return nil }
func (t *testModel) View() string  { return t.text.View() }
func (t *testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newM, newCmd := t.text.Update(msg)
	t.text = newM
	return t, newCmd
}
func (t *testModel) Debug() string { return t.text.Debug() }

func testCmd(m tea.Model, cmd string, args ...string) (bool, tea.Model, tea.Cmd, error) {
	t := m.(*testModel)
	switch cmd {
	case "focus":
		t.text.Focus()
	case "blur":
		t.text.Blur()
	case "clearline":
		t.text.ClearLine()
	case "movetobegin":
		t.text.MoveToBegin()
	case "movetoend":
		t.text.MoveToEnd()
	case "cursorright":
		n := 1
		if len(args) > 0 {
			i, err := strconv.Atoi(args[0])
			if err != nil {
				return true, t, nil, err
			}
			n = i
		}
		t.text.CursorRight(n)
	case "cursorstart":
		t.text.CursorStart()
	case "resetviewcursordown":
		t.text.ResetViewCursorDown()
	case "insert":
		input := strings.Join(args, " ")
		s, err := strconv.Unquote(input)
		if err != nil {
			return true, t, nil, err
		}
		t.text.InsertString(s)
	case "placeholder":
		input := strings.Join(args, " ")
		s, err := strconv.Unquote(input)
		if err != nil {
			return true, t, nil, err
		}
		t.text.Placeholder = s
	case "setvalue":
		input := strings.Join(args, " ")
		s, err := strconv.Unquote(input)
		if err != nil {
			return true, t, nil, err
		}
		t.text.SetValue(s)
	case "customprompt":
		t.text.SetPromptFunc(3, func(i int) string {
			switch i {
			case 0:
				return "@@>"
			case 3:
				return "! >"
			}
			return ">"
		})
	default:
		return false, t, nil, nil
	}
	return true, t, nil, nil
}
