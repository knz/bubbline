package textarea

import (
	"fmt"
	"io"
	"os"
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
			catwalk.WithObserver("curline", func(out io.Writer, m tea.Model) error {
				s := m.(*testModel).text.CurrentLine()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("props", func(out io.Writer, m tea.Model) error {
				t := &m.(*testModel).text
				fmt.Fprintf(out, "Focused: %v\n", t.Focused())
				fmt.Fprintf(out, "Width: %d, Height: %d, LogicalHeight: %d\n", t.Width(), t.Height(), t.LogicalHeight())
				fmt.Fprintf(out, "Length: %d, LineCount: %d, NumLinesInValue: %d\n",
					t.Length(), t.LineCount(), t.NumLinesInValue(),
				)
				return nil
			}),
			catwalk.WithObserver("pos", func(out io.Writer, m tea.Model) error {
				t := &m.(*testModel).text
				fmt.Fprintf(out, "Line: %d (row %d, col %d, lastCharOffset %d)\n",
					t.Line(), t.row, t.col, t.lastCharOffset,
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
	case "insert":
		input := strings.Join(args, " ")
		fmt.Fprintf(os.Stderr, "to unquote: %s", input)
		s, err := strconv.Unquote(input)
		if err != nil {
			return true, t, nil, err
		}
		t.text.InsertString(s)
	default:
		return false, t, nil, nil
	}
	return true, t, nil, nil
}
