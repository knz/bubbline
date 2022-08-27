package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/editline"
)

func main() {
	fmt.Println(`hello!

Enter some text below.

Try to autocomplete (tab) on the few letters at the beginning
of a name. Or just maybe one letter of the alphabet.
`)

	m := editline.New()

	// Configure the autocomplete function.
	m.AutoComplete = autocomplete

	for {
		m.Reset()
		if err := tea.NewProgram(m).Start(); err != nil {
			log.Fatal(err)
		}

		if m.Err != nil {
			if m.Err == io.EOF {
				// No more input.
				break
			}
			if errors.Is(m.Err, editline.ErrInterrupted) {
				// Entered Ctrl+C to cancel input.
				fmt.Println("^C")
			} else {
				fmt.Println("error:", m.Err)
			}
			continue
		}

		val := m.Value()
		fmt.Printf("\nYou have entered: %q\n", val)
	}
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

func autocomplete(
	v [][]rune, line, col int,
) (msg string, moveRight int, deleteLeft int, completions complete.Values) {
	// Detect the word under the cursor.
	word, wstart, wend := complete.FindWord(v, line, col)

	// Before the completion starts, move the cursor
	// that many positions to the right.
	moveRight = wend - col

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

	if len(candidates) == 1 {
		// Just one match. Fill that.
		completions.Prefill = candidates[0]
		deleteLeft = wend - wstart
	} else if len(candidates) > 1 {
		// More than one candidate. We will want to do two things
		// - pre-fill the longest common prefix in the input directly.
		// - present all the matches to the user as a menu.

		// Find longest common prefix and prefill that.
		// NB: this requires the candidates to be sorted
		// in alphabetical order already!
		completions.Prefill = complete.FindLongestCommonPrefix(
			candidates[0], candidates[len(candidates)-1],
			false /* case-sensitive */)
		deleteLeft = wend - wstart

		// Populate values to present to the user.
		completions.Categories = []string{"names"}
		completions.Completions = map[string][]string{"names": candidates}
	}

	// Note: moveRight is ignored if the switch above did not set
	// anything into the Prefill string.
	return msg, moveRight, deleteLeft, completions
}
