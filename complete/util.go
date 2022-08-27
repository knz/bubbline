package complete

import (
	"unicode"
	"unicode/utf8"
)

// FindWord is meant for use as a helper when implementing
// AutoComplete callbacks for the Model.AutoComplete field.
// Given AutoComplete's callback arguments, it searches
// and returns the start of the word that the cursor is currently
// on (as defined by the earliest character from the cursor
// that's not a whitespace) on the same line.
//
// NB: it does not cross line boundaries. The length in runes
// of the prefix from the cursor to the beginning of the word
// is guaranteed to be col-wordStart.
func FindWord(v [][]rune, line, col int) (word string, wordStart, wordEnd int) {
	curLine := v[line]
	curLen := len(curLine)
	if curLen == 0 {
		return "", 0, 0
	}
	// If the cursor is beyond the end of the line, move
	// it backwards once.
	if col >= curLen {
		col = curLen - 1
	}
	wordStart = col
	// Find beginning of word.
	for wordStart > 0 && !unicode.IsSpace(curLine[wordStart-1]) {
		wordStart--
	}
	wordEnd = col
	// Find end of word.
	for wordEnd <= curLen-1 && !unicode.IsSpace(curLine[wordEnd]) {
		wordEnd++
	}
	word = string(curLine[wordStart:wordEnd])
	return word, wordStart, wordEnd
}

// FindLongestCommonPrefix returns the longest common
// prefix between the two arguments.
func FindLongestCommonPrefix(first, last string, caseSensitive bool) string {
	en := len(first)
	if len(last) < en {
		en = len(last)
	}
	i := 0
	for {
		r, w := utf8.DecodeRuneInString(first[i:])
		l, _ := utf8.DecodeRuneInString(last[i:])
		if i >= en || (caseSensitive && r != l) || (!caseSensitive && unicode.ToUpper(r) != unicode.ToUpper(l)) {
			break
		}
		i += w
	}
	return first[:i]
}
