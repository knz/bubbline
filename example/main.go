package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/editline"
	"github.com/knz/bubbline/history"
)

var lore = regexp.MustCompile(`lo(\d+)$`)

var rwords = []string{
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

	m.KeyMap.Debug = key.NewBinding(key.WithKeys("ctrl+_"))
	m.Placeholder = "(your text here)"
	m.Prompt = "hello> "
	m.NextPrompt = "-> "
	m.AutoComplete = func(v [][]rune, line, col int) (msg string, consume int, extraInput []string) {
		p := col
		if p > 0 && p >= len(v[line]) {
			p = len(v[line]) - 1
		}
		if p > 0 && !unicode.IsSpace(v[line][p]) {
			// Find beginning of word.
			for p > 0 && !unicode.IsSpace(v[line][p-1]) {
				p--
			}
		}
		word := string(v[line][p:col])
		var complete []string
		const loremIpsum = ` ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.`
		if word == "lorem" {
			complete = []string{loremIpsum}
		} else if word == "hello" {
			complete = []string{" world"}
		} else if strings.ToLower(word) == "all" {
			complete = []string{" human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood."}
		} else if m := lore.FindStringSubmatch(word); m != nil {
			n, _ := strconv.Atoi(m[1])
			complete = []string{loremIpsum[:n]}
		} else {
			// Select r-words.
			wu := strings.ToUpper(word)
			complete = []string{""}
			for _, r := range rwords {
				if strings.HasPrefix(r, wu) {
					complete = append(complete, r)
				}
			}
			if len(complete) == 1 {
				// No match.
				complete = complete[:0]
				msg = fmt.Sprintf("autocomplete: ...%s\ntip: try completing after 'lorem' or 'r'", word)
			} else if len(complete) == 2 {
				// Just 1 match.
				complete[0] = complete[1]
				consume = col - p
				complete = complete[:1]
			} else {
				// Find longest common prefix.
				first, last := complete[1], complete[len(complete)-1]
				en := len(first)
				if len(last) < en {
					en = len(last)
				}
				i := 0
				for {
					r, w := utf8.DecodeRuneInString(first[i:])
					l, _ := utf8.DecodeRuneInString(last[i:])
					if i >= en || r != l {
						break
					}
					i += w
				}
				complete[0] = first[:i]
				consume = col - p
			}
		}
		return msg, consume, complete
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
