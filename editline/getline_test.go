package editline_test

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/bubbline"
	"github.com/knz/catwalk"
)

func TestBubbline(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if runtime.GOOS == "windows" && strings.HasSuffix(path, "job_control") {
			return
		}

		m := bubbline.New()
		catwalk.RunModel(t, path, m,
			catwalk.WithUpdater(testCmd),
			catwalk.WithObserver("value", func(out io.Writer, m tea.Model) error {
				s := m.(*bubbline.Editor).Value()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("err", func(out io.Writer, m tea.Model) error {
				e := m.(*bubbline.Editor).Err
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
	t := m.(*bubbline.Editor)
	switch cmd {
	case "unset_editor_env":
		os.Unsetenv("EDITOR")
	case "set_editor_env":
		os.Setenv("EDITOR", "invalid")
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
