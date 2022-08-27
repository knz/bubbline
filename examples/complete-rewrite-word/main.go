package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/editline"
)

func main() {
	fmt.Println(`hello!

Enter some text below.

Try to autocomplete (tab) on 'h', 'he', 'hel', 'hell', or 'hello'.
It's case-insensitive!
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
) (msg string, moveRight int, deleteLeft int, completions complete.Values) {
	// Detect the word under the cursor.
	word, wstart, wend := complete.FindWord(v, line, col)

	// Before the completion starts, move the cursor
	// that many positions to the right.
	moveRight = wend - col

	// Is this a part of the word "hello"?
	const specialWord = "HELLO"
	if strings.HasPrefix(specialWord, strings.ToUpper(word)) {
		// Yes: rewrite.
		completions.Prefill = specialWord
		deleteLeft = wend - wstart
	}

	// Note: moveRight is ignored if the switch above did not set
	// anything into the Prefill string.
	return msg, moveRight, deleteLeft, completions
}
