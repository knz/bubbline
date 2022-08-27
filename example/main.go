package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/editline"
	"github.com/knz/bubbline/history"
)

var lore = regexp.MustCompile(`lo(\d+)$`)

var rnames = []string{
	"Reemst",
	"rapster",
	"Robbertsen",
	"ruggenwervels",
	"reisverenigingen",
	"radiojournalisten",
	"registratiebureau",
	"Rashid",
	"rectoscoop",
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
Blocks of input are terminated with a semicolon.`)

	h, err := history.LoadHistory("test.history")
	if err != nil {
		fmt.Println("history load error:", err)
	}
	m := editline.New()

	m.ReflowFn = func(x bool, y string, _ int) (bool, string, string) {
		return editline.DefaultReflow(x, y, 72)
	}
	m.KeyMap.Debug = key.NewBinding(key.WithKeys("ctrl+_", "ctrl+@"))
	m.Placeholder = "(your text here)"
	m.Prompt = "hello> "
	m.NextPrompt = "-> "
	m.AutoComplete = func(v [][]rune, line, col int) (msg string, moveRight, deleteLeft int, completions complete.Values) {
		word, wstart, wend := complete.FindWord(v, line, col)
		moveRight = wend - col
		const loremIpsum = ` ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.`
		if word == "lorem" {
			completions.Prefill = loremIpsum
		} else if word == "hello" {
			completions.Prefill = " world"
		} else if strings.ToLower(word) == "all" {
			completions.Prefill = " human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood."
		} else if m := lore.FindStringSubmatch(word); m != nil {
			n, _ := strconv.Atoi(m[1])
			completions.Prefill = loremIpsum[:n]
		} else {
			msg = fmt.Sprintf("autocomplete: ...%q (%d %d %d)\ntip: try completing after 'lorem' or 'r'", word, wstart, wend, col)
			// Select r-words.
			var comps []string
			var rcomp []string
			var acomp []string
			wu := strings.ToUpper(word)
			for _, r := range rnames {
				if strings.HasPrefix(strings.ToUpper(r), wu) {
					comps = append(comps, r)
					acomp = append(acomp, r)
				}
			}
			for _, r := range rwords {
				if strings.HasPrefix(strings.ToUpper(r), wu) {
					comps = append(comps, r)
					rcomp = append(rcomp, r)
				}
			}
			sort.Slice(comps, func(i, j int) bool { return strings.ToLower(comps[i]) < strings.ToLower(comps[j]) })
			if len(comps) == 0 {
				// No match.
			} else if len(comps) == 1 {
				// Just 1 match.
				completions.Prefill = comps[0]
				deleteLeft = wend - wstart
			} else {
				// Find longest common prefix.
				completions.Prefill = complete.FindLongestCommonPrefix(comps[0], comps[len(comps)-1], false)
				deleteLeft = wend - wstart
				// Populate values.
				completions.Completions = map[string][]string{}
				if len(acomp) > 0 {
					completions.Categories = append(completions.Categories, "words")
					completions.Completions["words"] = acomp
				}
				if len(rcomp) > 0 {
					completions.Categories = append(completions.Categories, "keywords")
					completions.Completions["keywords"] = rcomp
				}
			}
		}
		return msg, moveRight, deleteLeft, completions
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
			fmt.Println("\nEnter more input:")
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
