package main

import (
	"errors"
	"fmt"
	"io"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/editline"
	"github.com/knz/bubbline/history"
)

func main() {
	fmt.Println(`hello!`)

	h, err := history.LoadHistory("test.history")
	if err != nil {
		fmt.Println("history load error:", err)
	}

	m := editline.New()
	m.SetHistory(h)

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
		m.AddHistoryEntry(val)

		// Auto-save history.
		err := history.SaveHistory(m.GetHistory(), "test.history")
		if err != nil {
			fmt.Println("history save error:", err)
		}
	}
}
