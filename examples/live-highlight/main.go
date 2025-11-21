package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/highlight"
)

// simpleHighlighter is an example implementation of the highlight.Highlighter interface.
// It provides live highlighting for a few keywords.
type simpleHighlighter struct {
	// Define complete styles for each token type.
	styleKeyword lipgloss.Style
	styleNumber  lipgloss.Style
	styleDefault lipgloss.Style
}

// newSimpleHighlighter creates our highlighter with its styles pre-defined.
func newSimpleHighlighter() *simpleHighlighter {
	return &simpleHighlighter{
		styleKeyword: lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true), // Blue and Bold
		styleNumber:  lipgloss.NewStyle().Foreground(lipgloss.Color("35")),            // Magenta
		styleDefault: lipgloss.NewStyle(),                                             // Use the terminal's default foreground color
	}
}

// Highlight tokenizes the line and applies styles.
func (h *simpleHighlighter) Highlight(line string) []highlight.Token {
	var tokens []highlight.Token

	// This logic preserves spaces by finding words and the gaps between them.
	var lastPos int
	for _, word := range strings.Fields(line) {
		idx := strings.Index(line[lastPos:], word)
		// Add the whitespace before the word as a plain token.
		if idx > 0 {
			tokens = append(tokens, highlight.Token{
				Value: line[lastPos : lastPos+idx],
				Style: h.styleDefault,
			})
		}

		// --- Start with the default style for every word ---
		finalStyle := h.styleDefault

		// --- Apply a specific style only if it matches a category ---
		upperWord := strings.ToUpper(word)
		if upperWord == "SELECT" || upperWord == "FROM" || upperWord == "WHERE" {
			finalStyle = h.styleKeyword
		} else if _, err := strconv.Atoi(word); err == nil {
			finalStyle = h.styleNumber
		}

		// Add the word itself with its determined style.
		tokens = append(tokens, highlight.Token{
			Value: word,
			Style: finalStyle,
		})
		lastPos += idx + len(word)
	}
	// Add any trailing whitespace.
	if lastPos < len(line) {
		tokens = append(tokens, highlight.Token{
			Value: line[lastPos:],
			Style: h.styleDefault,
		})
	}

	return tokens
}

func main() {
	fmt.Println("Live highlighter example. Type 'SELECT' or 'FROM' to see live highlighting. Ctrl+D to exit.")

	m := bubbline.New()

	// 1. Instantiate our highlighter using the constructor.
	highlighter := newSimpleHighlighter()

	// 2. Set it on the bubbline editor instance.
	m.SetHighlighter(highlighter)

	// 3. Run the editor.
	for {
		val, err := m.GetLine()

		if err == io.EOF {
			fmt.Println("\nBye!")
			break
		}
		if err != nil {
			fmt.Println("error:", err)
			break
		}
		fmt.Printf("\nYou entered: %q\n", val)
	}
}
