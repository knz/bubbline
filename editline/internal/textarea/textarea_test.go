package textarea

import (
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
		catwalk.RunModel(t, path, &m, catwalk.WithUpdater(testCmd))
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
	default:
		return false, t, nil, nil
	}
	return true, t, nil, nil
}
