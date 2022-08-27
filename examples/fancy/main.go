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
	"github.com/knz/bubbline/editline"
)

func main() {
	fmt.Println(`hello!

Input ends automatically on semicolon.
Try autocompleting on 'lorem', 'all', 'hello', 'lo' followed by digits,
or the letter 'r'.
`)

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

func autocomplete(
	v [][]rune, line, col int,
) (msg string, moveRight int, deleteLeft int, completions complete.Values) {
	// Detect the word under the cursor.
	word, wstart, wend := complete.FindWord(v, line, col)

	// Before the completion starts, move the cursor
	// that many positions to the right.
	moveRight = wend - col

	// Just an informational message to display at the top.
	// This is optional!
	msg = fmt.Sprintf("We're matching %q!", word)

	// Try to complete the simple words first.
	switch word {
	case "lorem":
		completions.Prefill = loremIpsum
	case "hello":
		completions.Prefill = " world"
	case "all":
		completions.Prefill = firstArticle
	default:
		// Does the word match the string "lo" followed by digits?
		if m := lore.FindStringSubmatch(word); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n > len(loremIpsum) {
				n = len(loremIpsum)
			}
			completions.Prefill = loremIpsum[:n]
		}
	}

	if completions.Prefill == "" {
		// No luck so far? Try harder.

		// Where we will collect the candidates.
		candidatesPerCategory := map[string][]string{}
		var allCandidates []string

		// We're going to match the word lowercase.
		lword := strings.ToLower(word)

		// Is the word the start of a name?
		for _, name := range names {
			if strings.HasPrefix(strings.ToLower(name), lword) {
				allCandidates = append(allCandidates, name)
				candidatesPerCategory["name"] = append(candidatesPerCategory["name"], name)
			}
		}
		// Is the word the start of a keyword?
		for _, kw := range keywords {
			if strings.HasPrefix(strings.ToLower(kw), lword) {
				allCandidates = append(allCandidates, kw)
				candidatesPerCategory["keywords"] = append(candidatesPerCategory["keywords"], kw)
			}
		}
		// Is the word the start of a Dutch word?
		for _, dw := range dutchWords {
			if strings.HasPrefix(strings.ToLower(dw), lword) {
				allCandidates = append(allCandidates, dw)
				candidatesPerCategory["Dutch"] = append(candidatesPerCategory["Dutch"], dw)
			}
		}

		if len(allCandidates) == 1 {
			// Just one match. Fill that.
			completions.Prefill = allCandidates[0]
			deleteLeft = wend - wstart
		} else if len(allCandidates) > 1 {
			// More than one candidate. We will want to do two things
			// - pre-fill the longest common prefix in the input directly.
			// - present all the matches to the user as a menu.

			// Find longest common prefix and prefill that.
			// NB: this requires the candidates to be sorted
			// in alphabetical order already!
			sort.Slice(allCandidates, func(i, j int) bool {
				return strings.ToLower(allCandidates[i]) < strings.ToLower(allCandidates[j])
			})
			completions.Prefill = complete.FindLongestCommonPrefix(
				allCandidates[0], allCandidates[len(allCandidates)-1],
				false /* case-sensitive */)
			deleteLeft = wend - wstart

			// Populate values to present to the user.
			for k := range candidatesPerCategory {
				completions.Categories = append(completions.Categories, k)
			}
			sort.Strings(completions.Categories)
			completions.Completions = candidatesPerCategory
		}
	}

	// Note: moveRight is ignored if the switch above did not set
	// anything into the Prefill string.
	return msg, moveRight, deleteLeft, completions
}

var lore = regexp.MustCompile(`lo(\d+)$`)

const loremIpsum = ` ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.`

const firstArticle = ` human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood.`

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
