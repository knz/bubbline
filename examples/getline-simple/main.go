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
			} else if errors.Is(err, bubbline.ErrTerminated) {
				fmt.Println("terminated")
				break
			} else {
				fmt.Println("error:", err)
			}
			continue
		}

		fmt.Printf("\nYou have entered: %q\n", val)
	}
}
