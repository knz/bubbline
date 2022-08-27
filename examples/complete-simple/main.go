package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/editline"
)

func main() {
	fmt.Println(`hello!

Enter some text below.

Try to autocomplete (tab) after 'hello', 'all', 'lorem' or 'lo'
followed by a few digits (e.g. 'lo30').

You can also press tab with the cursor in the middle of a word!
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

func autocomplete(
	v [][]rune, line, col int,
) (msg string, moveRight int, _ int, completions complete.Values) {
	// Detect the word under the cursor.
	word, _, wend := complete.FindWord(v, line, col)

	// Just an informational message to display at the top.
	// This is optional!
	msg = fmt.Sprintf("We're matching %q!", word)

	// Before the completion starts, move the cursor
	// that many positions to the right.
	moveRight = wend - col

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

	// Note: moveRight is ignored if the switch above did not set
	// anything into the Prefill string.
	return msg, moveRight, 0, completions
}

var lore = regexp.MustCompile(`lo(\d+)$`)

const loremIpsum = ` ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.`

const firstArticle = ` human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood.`
