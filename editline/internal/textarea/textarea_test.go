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
	m := testModel{
		text: New(),
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		catwalk.RunModel(t, path, &m,
			catwalk.WithUpdater(testCmd),
			catwalk.WithObserver("value", func(out io.Writer, m tea.Model) error {
				s := m.(*testModel).text.Value()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("props", func(out io.Writer, m tea.Model) error {
				t := &m.(*testModel).text
				fmt.Fprintf(out, "Focused: %v\n", t.Focused())
				fmt.Fprintf(out, "Width: %d, Height: %d, LogicalHeight: %d\n", t.Width(), t.Height(), t.LogicalHeight())
				fmt.Fprintf(out, "Length: %d, LineCount: %d, Line: %d (row %d, col %d, lastCharOffset %d)\n",
					t.Length(), t.LineCount(), t.Line(), t.row, t.col, t.lastCharOffset,
				)
				fmt.Fprintf(out, "LineInfo: %+v\n", t.LineInfo())
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

func testCmd(m tea.Model, cmd string, args ...string) (bool, tea.Model, tea.Cmd, error) {
	t := m.(*testModel)
	switch cmd {
	case "focus":
		t.text.Focus()
	case "blur":
		t.text.Blur()
	case "insert":
		s, err := strconv.Unquote(strings.Join(args, " "))
		if err != nil {
			return true, t, nil, err
		}
		t.text.InsertString(s)
	default:
		return false, t, nil, nil
	}
	return true, t, nil, nil
}
