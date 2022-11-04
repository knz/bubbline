package bubbline

import (
	"fmt"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
)

func TestBubbline(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		m := New()
		catwalk.RunModel(t, path, m,
			catwalk.WithUpdater(testCmd),
			catwalk.WithObserver("value", func(out io.Writer, m tea.Model) error {
				s := m.(*Editor).Value()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("err", func(out io.Writer, m tea.Model) error {
				e := m.(*Editor).Err
				if e != nil {
					fmt.Fprintf(out, "%v", e)
				} else {
					fmt.Fprintf(out, "<no error>")
				}
				return nil
			}),
		)
	})
}

func testCmd(m tea.Model, cmd string, args ...string) (bool, tea.Model, tea.Cmd, error) {
	t := m.(*Editor)
	switch cmd {
	case "reset":
		t.Reset()
	case "focus":
		t.Focus()
	case "blur":
		t.Blur()
	case "enable_ext_edit":
		t.SetExternalEditorEnabled(true, "hello")
	default:
		return false, t, nil, nil
	}
	return true, t, nil, nil
}
