package main

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/knz/bubbline"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/computil"
	"github.com/knz/bubbline/editline"
)

func main() {
	fmt.Println(`hello!

Input ends automatically on semicolon.
Try autocompleting on 'lorem', 'all', 'hello', 'lo' followed by digits,
or the letter 'r'.`)
	fmt.Println()

	m := bubbline.New()

	// Initial input.
	m.Placeholder = "(your text here)"
	m.Prompt = "hello> "
	m.NextPrompt = "-> "

	// Clamp reflow at 72 columns. Easier to demo reflow on lorem ipsum
	// this way.
	m.Reflow = func(x bool, y string, _ int) (bool, string, string) {
		return editline.DefaultReflow(x, y, 72)
	}

	// Configure autocomplete function.
	m.AutoComplete = autocomplete

	// End input on semicolon automatically.
	m.CheckInputComplete = func(v [][]rune, line, col int) bool {
		if line == len(v)-1 && // Enter on last row.
			strings.HasSuffix(string(v[len(v)-1]), ";") { // Semicolon at end of last row.
			return true
		}
		return false
	}

	// Load and configure history.
	if err := m.LoadHistory("test.history"); err != nil {
		fmt.Println("history load error:", err)
	}
	m.SetAutoSaveHistory("test.history", true)

	// Enable debug mode.
	m.SetDebugEnabled(true)

	// Enable external editor.
	m.SetExternalEditorEnabled(true, "sql")

	// Read-print loop starts here.
	for {
		val, err := m.GetLine()

		if err != nil {
			if err == io.EOF {
				// No more input.
				break
			}
			if errors.Is(err, bubbline.ErrInterrupted) {
				// Entered Ctrl+C to cancel input.
				fmt.Println("^C")
			} else {
				fmt.Println("error:", err)
			}
			continue
		}

		fmt.Printf("\nYou have entered: %q\n", val)
		m.AddHistory(val)
	}
}

func autocomplete(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	// Detect the word under the cursor.
	word, wstart, wend := computil.FindWord(v, line, col)

	// Just an informational message to display at the top.
	// This is optional!
	msg = fmt.Sprintf("We're matching %q!", word)

	// Try to complete the simple words first.
	switch word {
	case "lorem":
		completions = editline.SingleWordCompletion(loremIpsum, col, wstart, wend)
	case "hello":
		completions = editline.SingleWordCompletion("hello world", col, wstart, wend)
	case "all":
		completions = editline.SingleWordCompletion(firstArticle, col, wstart, wend)
	default:
		// Does the word match the string "lo" followed by digits?
		if m := lore.FindStringSubmatch(word); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n > len(loremIpsum) {
				n = len(loremIpsum)
			}
			completions = editline.SingleWordCompletion(loremIpsum[:n], col, wstart, wend)
		}
	}

	if completions == nil {
		// No luck so far? Try harder.

		// Where we will collect the candidates.
		candidatesPerCategory := map[string][]string{}

		// We're going to match the word lowercase.
		lword := strings.ToLower(word)

		numCandidates := 0

		// Is the word the start of a name?
		for _, name := range names {
			if strings.HasPrefix(strings.ToLower(name), lword) {
				candidatesPerCategory["name"] = append(candidatesPerCategory["name"], name)
				numCandidates++
			}
		}
		// Is the word the start of a keyword?
		for _, kw := range keywords {
			if strings.HasPrefix(strings.ToLower(kw), lword) {
				candidatesPerCategory["keywords"] = append(candidatesPerCategory["keywords"], kw)
				numCandidates++
			}
		}
		// Is the word the start of a Dutch word?
		for _, dw := range dutchWords {
			if strings.HasPrefix(strings.ToLower(dw), lword) {
				candidatesPerCategory["Dutch"] = append(candidatesPerCategory["Dutch"], dw)
				numCandidates++
			}
		}
		completions = &multiComplete{
			Values:     complete.MapValues(candidatesPerCategory, nil),
			moveRight:  wend - col,
			deleteLeft: wend - wstart,
		}
	}

	// Note: moveRight is ignored if the switch above did not set
	// anything into the Prefill string.
	return msg, completions
}

type multiComplete struct {
	complete.Values
	moveRight, deleteLeft int
}

func (m *multiComplete) Candidate(e complete.Entry) editline.Candidate {
	return candidate{e.Title(), m.moveRight, m.deleteLeft}
}

type candidate struct {
	repl                  string
	moveRight, deleteLeft int
}

func (m candidate) Replacement() string { return m.repl }
func (m candidate) MoveRight() int      { return m.moveRight }
func (m candidate) DeleteLeft() int     { return m.deleteLeft }

var lore = regexp.MustCompile(`lo(\d+)$`)

const loremIpsum = `lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.`

const firstArticle = `all human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood.`

var dutchWords = []string{
	"Reemst",
	"rapster",
	"Robbertsen",
	"ruggenwervels",
	"reisverenigingen",
	"radiojournalisten",
	"registratiebureau",
	"Rashid",
	"rondzweef",
	"respondeerde",
	"reuzig",
	"relatiebureau",
	"Reukers",
	"rails",
	"Reith",
	"ripper",
	"respecteerden",
	"routeplan",
	"renster",
	"regeringswisselingen",
	"rechterschouder",
	"ruste",
	"Rebers",
	"rokerskamer",
	"rotseilandjes",
	"rondden",
}

var keywords = []string{
	"RANGE",
	"RANGES",
	"READ",
	"REASON",
	"REASSIGN",
	"RECURRING",
	"RECURSIVE",
	"REF",
	"REFRESH",
	"REGION",
	"REGIONAL",
	"REGIONS",
	"REINDEX",
	"RELATIVE",
	"RELEASE",
	"RELOCATE",
	"RENAME",
	"REPEATABLE",
	"REPLACE",
	"REPLICATION",
	"RESET",
	"RESTART",
	"RESTORE",
	"RESTRICT",
	"RESTRICTED",
	"RESUME",
	"RETRY",
	"RETURN",
	"RETURNS",
	"REVISION_HISTORY",
	"REVOKE",
	"ROLE",
	"ROLES",
	"ROLLBACK",
	"ROLLUP",
	"ROUTINES",
	"ROWS",
	"RULE",
	"RUNNING",
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
