package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/editline"
	"github.com/knz/bubbline/history"
)

func main() {
	fmt.Println(`hello!
Enter some text below.
Blocks of input are terminated with a semicolon.
Press Ctrl+C to interrupt; Ctrl+D to terminate.`)

	h, err := history.LoadHistory("test.history")
	if err != nil {
		fmt.Println("history load error:", err)
	}
	m := editline.New()

	m.AutoComplete = func(v string, cursor int) (msg, extraInput string) {
		p := cursor - 5
		if p < 0 {
			p = 0
		}
		msg = fmt.Sprintf("autocomplete: ...%s", v[p:cursor])
		return msg, "<TAB>"
	}

	m.CheckInputComplete = func(v string, row int) bool {
		vs := strings.Split(v, "\n")
		if row == len(vs)-1 && // Enter on last row.
			strings.HasSuffix(vs[len(vs)-1], ";") { // Semicolon on last row.
			return true
		}
		return false
	}

	m.SetHistory(h)

	atStart := true
	for {
		if atStart {
			atStart = false
		} else {
			fmt.Println("\nEnter more input (Ctrl+D to terminate):")
		}
		p := tea.NewProgram(m)
		m.Reset()
		if err := p.Start(); err != nil {
			log.Fatal(err)
		}

		if m.Err != nil {
			if m.Err == io.EOF {
				break
			}
			if errors.Is(m.Err, editline.ErrInterrupted) {
				fmt.Println("^C")
			} else {
				fmt.Println("error: %v", m.Err)
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
