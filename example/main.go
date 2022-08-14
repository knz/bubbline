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

	m.Prompt = "hello> "
	m.NextPrompt = "-> "
	m.AutoComplete = func(v [][]rune, line, col int) (msg, extraInput string) {
		p := col - 5
		if p < 0 {
			p = 0
		}
		word := string(v[line][p:col])
		msg = fmt.Sprintf("autocomplete: ...%s", word)
		complete := ""
		if word == "lorem" {
			complete = ` ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.`
		} else if word == "hello" {
			complete = " world"
		} else if strings.ToLower(word) == "all h" {
			complete = "uman beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood."
		} else {
			msg += "\ntip: try completing after 'lorem'"
		}
		return msg, complete
	}

	m.CheckInputComplete = func(v [][]rune, line, col int) bool {
		if line == len(v)-1 && // Enter on last row.
			strings.HasSuffix(string(v[len(v)-1]), ";") { // Semicolon at end of last row.
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
