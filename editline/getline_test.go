package editline_test

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/computil"
	"github.com/knz/bubbline/editline"
	"github.com/knz/catwalk"
	"github.com/muesli/termenv"
)

// TestBubbline tests the bubbline widget and the Editor API.
// We place this test in this package, and not the top level
// package, so that its results can be fed into coveralls.io.
// (Coverage test results are per-package.)
func TestBubbline(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if runtime.GOOS == "windows" && strings.HasSuffix(path, "job_control") {
			return
		}

		m := bubbline.New()

		// Ensure the cursor is visible in the test outputs. We want this
		// because we want to check that event processing positions the
		// cursor correctly.
		lipgloss.SetColorProfile(termenv.ANSI)

		catwalk.RunModel(t, path, m,
			catwalk.WithUpdater(testCmd),
			catwalk.WithObserver("value", func(out io.Writer, m tea.Model) error {
				s := m.(*bubbline.Editor).Value()
				fmt.Fprintf(out, "%q", s)
				return nil
			}),
			catwalk.WithObserver("history", func(out io.Writer, m tea.Model) error {
				h := m.(*bubbline.Editor).GetHistory()
				for _, e := range h {
					fmt.Fprintf(out, "%s\n", e)
				}
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
	case "set_history":
		t.SetHistory([]string{
			// Note: we care about the position of the word
			//   "world"
			// in these example history entries:
			//
			// - one must be at the very end of a history entry.  This is
			//   needed to check that the pattern match also works at the
			//   end of lines.
			//
			// - there should be one entry without the word in-between two
			//   entries that have it, to check that the search skips over
			//   entries.
			"say hello to the world",
			"peter parker was not spiderman",
			"this is a big world indeed",
		})
	case "add_history":
		t.AddHistoryEntry(t.Value())
	case "toggle_dedup_history":
		t.DedupHistory = !t.DedupHistory
	case "limit_history_size":
		t.MaxHistorySize = 2
	case "unset_editor_env":
		os.Unsetenv("EDITOR")
	case "set_editor_env":
		os.Setenv("EDITOR", "invalid")
	case "noop":
	case "reset":
		t.Reset()
	case "focus":
		t.Focus()
	case "blur":
		t.Blur()
	case "enable_ext_edit":
		t.SetExternalEditorEnabled(true, "hello")
	case "enable_debug":
		t.SetDebugEnabled(true)
	case "disable_debug":
		t.SetDebugEnabled(false)
	case "clear_reflow":
		t.Reflow = nil
	case "configure_check_eof":
		t.CheckInputComplete = func(e [][]rune, line, col int) bool {
			for _, l := range e {
				for _, c := range l {
					if c == '.' {
						return true
					}
				}
			}
			return false
		}
	case "set_autocomplete_1":
		t.AutoComplete = autocomplete1
	case "set_autocomplete_2":
		t.AutoComplete = autocomplete2
	case "show_cursor":
		t.CursorMode = cursor.CursorStatic
	case "hide_cursor":
		t.CursorMode = cursor.CursorHide
	case "limit_max_width":
		t.MaxWidth = 10
	case "limit_max_height":
		t.MaxHeight = 3
	default:
		return false, t, nil, nil
	}
	return true, t, nil, nil
}

func autocomplete1(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	// Detect the word under the cursor.
	word, wstart, wend := computil.FindWord(v, line, col)

	// Just an informational message to display at the top.
	// This is optional!
	msg = fmt.Sprintf("We're matching %q!", word)

	switch word {
	case "hello":
		return msg, editline.SingleWordCompletion("hello world", col, wstart, wend)
	default:
		return msg, nil
	}
}

func autocomplete2(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	// Detect the word under the cursor.
	word, wstart, wend := computil.FindWord(v, line, col)

	var titleWord string
	if len(word) > 0 {
		titleWord = strings.ToTitle(word[:1]) + strings.ToLower(word[1:])
	}

	// Is this a part of a name?
	var candidates []string
	for _, name := range names {
		if strings.HasPrefix(name, titleWord) {
			// Yes: add the matching name to the list of candidate
			// completions.
			candidates = append(candidates, name)
		}
	}

	// Just an informational message to display at the top.
	// This is optional!
	msg = fmt.Sprintf("We're matching %q!", titleWord)

	if len(candidates) == 0 {
		return msg, nil
	}

	return msg, editline.SimpleWordsCompletion(candidates, "names", col, wstart, wend)
}

var names = func() []string {
	s := []string{"Andrew", "Anthony", "Arthur", "Brian", "Carl",
		"Charles", "Christopher", "Daniel", "David", "Dennis", "Donald",
		"Douglas", "Edward", "Eric", "Frank", "Gary", "George", "Gregory",
		"Harold", "Henry", "Jack", "James", "Jason", "Jeffrey", "Jerry",
		"John", "Jose", "Joseph", "Joshua", "Kenneth", "Kevin", "Larry",
		"Mark", "Matthew", "Michael", "Patrick", "Paul", "Peter", "Raymond",
		"Richard", "Robert", "Ronald", "Ryan", "Scott", "Stephen", "Steven",
		"Thomas", "Timothy", "Walter", "William", "Alice", "Amanda", "Amy",
		"Angela", "Ann", "Anna", "Barbara", "Betty", "Brenda", "Caroline",
		"Catherine", "Christine", "Cynthia", "Deborah", "Debra", "Diane",
		"Donna", "Dorothy", "Elizabeth", "Emily", "Frances", "Helen", "Janet",
		"Jennifer", "Jessica", "Joyce", "Karen", "Kathleen", "Kimberly",
		"Laura", "Linda", "Lisa", "Margaret", "Maria", "Marie", "Martha",
		"Mary", "Michelle", "Nancy", "Pamela", "Patricia", "Rebecca", "Ruth",
		"Sandra", "Sarah", "Sharon", "Shirley", "Stephanie", "Susan",
		"Virginia"}
	sort.Strings(s)
	return s
}()
