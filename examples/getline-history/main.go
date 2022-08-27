package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/knz/bubbline"
)

func main() {
	fmt.Println(`hello!`)

	m := bubbline.New()

	if err := m.LoadHistory("test.history"); err != nil {
		fmt.Println("history load error:", err)
	}
	m.SetAutoSaveHistory("test.history", true)

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
