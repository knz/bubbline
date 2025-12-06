package highlight

import "github.com/charmbracelet/lipgloss"

// Token represents a single styled segment of text.
type Token struct {
	Value string
	Style lipgloss.Style
}

// Highlighter is the interface for a syntax highlighter that
// can tokenize a line of text.
type Highlighter interface {
	// Highlight takes a line of text and returns a slice of styled Tokens.
	Highlight(line string) []Token
}
